package main

import (
	"context"
	"fmt"
	"net"
	"os"
	"strings"
	"sync"
	"time"

	"google.golang.org/grpc"
	"k8s.io/klog/v2"
	pluginapi "k8s.io/kubelet/pkg/apis/deviceplugin/v1beta1"
)

// FakeGPU 实现 pluginapi.DevicePluginServer (Device Plugin v1beta1).
//
// 与 ../device-plugin demo 的关键差异是 Allocate 的返回值:
//   - 学 NVIDIA k8s-device-plugin: 仅返回 envs, 不返回 Mounts / Devices.
//   - NVIDIA_VISIBLE_DEVICES 是 nvidia-container-runtime 的 prestart hook 真正读取的 env,
//     hook 据此把对应 /dev/nvidiaX + cuda 库 bind-mount 进容器.
//   - 在没有真实 NVIDIA runtime 的集群上, 这个 env 仍会被注入容器,
//     用户 kubectl logs 就能看到, 整条注入链路是清晰可观察的.
type FakeGPU struct {
	pluginapi.UnimplementedDevicePluginServer

	devices []*pluginapi.Device

	server *grpc.Server
	mu     sync.Mutex
	stopCh chan struct{}
}

func newFakeGPU(n int) *FakeGPU {
	var devs []*pluginapi.Device
	for i := 0; i < n; i++ {
		devs = append(devs, &pluginapi.Device{
			// NVIDIA 风格的 UUID 字符串. 真实 NVIDIA plugin 通过 NVML 拿到
			// 形如 GPU-3a9b4d5e-... 的 UUID; 这里用全零 + index 的伪造形式,
			// 既保留 "GPU-" 前缀让链路日志看起来像真的, 又保证可复现.
			ID:     fmt.Sprintf("GPU-00000000-0000-0000-0000-00000000000%d", i),
			Health: pluginapi.Healthy,
		})
	}
	return &FakeGPU{devices: devs, stopCh: make(chan struct{})}
}

// serve 在 pluginSock 上启动 gRPC server, 并向其注册 DevicePluginServer.
func (p *FakeGPU) serve() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if err := os.Remove(pluginSock); err != nil && !os.IsNotExist(err) {
		return err
	}
	lis, err := net.Listen("unix", pluginSock)
	if err != nil {
		return err
	}
	p.server = grpc.NewServer()
	pluginapi.RegisterDevicePluginServer(p.server, p)
	p.stopCh = make(chan struct{})

	go func() {
		klog.Infof("fake-gpu plugin listening on %s", pluginSock)
		if err := p.server.Serve(lis); err != nil {
			klog.Errorf("grpc serve exited: %v", err)
		}
	}()

	// 等 server 真正起来再返回, 否则后续 Register 调用 kubelet 时,
	// kubelet 反向 dial 我们这边会失败.
	return waitForSocket(pluginSock, 5*time.Second)
}

// stop 优雅停止 gRPC server.
func (p *FakeGPU) stop() {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.server != nil {
		p.server.GracefulStop()
		p.server = nil
	}
	select {
	case <-p.stopCh:
	default:
		close(p.stopCh)
	}
}

// GetDevicePluginOptions 由 kubelet 在 Register 后调用一次, 协商可选能力.
// 真实 NVIDIA plugin 这里会返回 GetPreferredAllocationAvailable=true 以支持
// NVLink/NUMA 拓扑感知; 本 demo 简化为 false.
func (p *FakeGPU) GetDevicePluginOptions(_ context.Context, _ *pluginapi.Empty) (*pluginapi.DevicePluginOptions, error) {
	return &pluginapi.DevicePluginOptions{
		PreStartRequired:                false,
		GetPreferredAllocationAvailable: false,
	}, nil
}

// ListAndWatch 上报当前 GPU 列表与健康状态. kubelet 据此更新
// Node.status.capacity["fake-gpu.k8s.io/gpu"] = 4.
// 真实 NVIDIA plugin 会用 NVML 周期巡检 XID / ECC, 不健康时移出列表.
func (p *FakeGPU) ListAndWatch(_ *pluginapi.Empty, srv pluginapi.DevicePlugin_ListAndWatchServer) error {
	// 立即发一次全量.
	if err := srv.Send(&pluginapi.ListAndWatchResponse{Devices: p.devices}); err != nil {
		return err
	}
	klog.Infof("ListAndWatch: pushed %d fake GPUs", len(p.devices))

	// 进入阻塞循环. Demo 中设备永远健康, 仅在 ticker 上"保活心跳"重发一次列表.
	ticker := time.NewTicker(60 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-srv.Context().Done():
			return nil
		case <-p.stopCh:
			return nil
		case <-ticker.C:
			if err := srv.Send(&pluginapi.ListAndWatchResponse{Devices: p.devices}); err != nil {
				return err
			}
		}
	}
}

// Allocate 把 kubelet 选中的若干 GPU UUID 转换为"注入容器的合同".
//
// 设计要点 (学 NVIDIA k8s-device-plugin):
//   - 仅返回 Envs, 不返回 Devices / Mounts.
//   - 在真实 NVIDIA 部署里, nvidia-container-runtime 的 prestart hook 读
//     NVIDIA_VISIBLE_DEVICES env, 把对应 /dev/nvidiaX + driver 库注入容器.
//   - 本 demo 没有真实 runtime, 但 env 仍会被 kubelet 合并进 CRI CreateContainer,
//     用户 kubectl logs 能看到, 直接证明"插件 -> kubelet -> CRI -> 容器"链路.
func (p *FakeGPU) Allocate(_ context.Context, req *pluginapi.AllocateRequest) (*pluginapi.AllocateResponse, error) {
	resp := &pluginapi.AllocateResponse{}
	for _, creq := range req.ContainerRequests {
		klog.Infof("Allocate: container requests GPUs=%v", creq.DevicesIDs)
		ids := strings.Join(creq.DevicesIDs, ",")
		car := &pluginapi.ContainerAllocateResponse{
			Envs: map[string]string{
				// 真实 NVIDIA runtime 读这个 env 来决定挂哪些 /dev/nvidiaX 和 driver 库.
				"NVIDIA_VISIBLE_DEVICES": ids,
				// 控制 hook 注入哪些 capability 集合 (compute = CUDA runtime, utility = nvidia-smi).
				"NVIDIA_DRIVER_CAPABILITIES": "compute,utility",
				// 同步给一个我们自己的 env, 便于演示"非 NVIDIA-aware 镜像也能看到分配结果".
				"FAKE_GPU_DEVICES":   ids,
				"FAKE_RESOURCE_NAME": resourceName,
			},
			// 注意: 故意不放 Mounts / Devices, 完全靠 env 注入,
			// 跟 NVIDIA Device Plugin 的默认行为一致.
			Annotations: map[string]string{
				"fake-gpu.k8s.io/allocated-uuids": ids,
			},
		}
		resp.ContainerResponses = append(resp.ContainerResponses, car)
	}
	return resp, nil
}

// PreStartContainer 由 kubelet 在容器启动前调用 (仅当 GetDevicePluginOptions
// 中 PreStartRequired=true). 本 demo 不需要, 直接返回空.
func (p *FakeGPU) PreStartContainer(_ context.Context, _ *pluginapi.PreStartContainerRequest) (*pluginapi.PreStartContainerResponse, error) {
	return &pluginapi.PreStartContainerResponse{}, nil
}

// GetPreferredAllocation 由 kubelet 在 DeviceManager 准备分配时调用 (可选).
// 本 demo 关闭该能力, 留空兜底实现.
func (p *FakeGPU) GetPreferredAllocation(_ context.Context, _ *pluginapi.PreferredAllocationRequest) (*pluginapi.PreferredAllocationResponse, error) {
	return &pluginapi.PreferredAllocationResponse{}, nil
}

// waitForSocket 轮询直到指定 socket 可以被 dial, 避免 race.
func waitForSocket(sock string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		conn, err := net.DialTimeout("unix", sock, time.Second)
		if err == nil {
			_ = conn.Close()
			return nil
		}
		time.Sleep(100 * time.Millisecond)
	}
	return fmt.Errorf("timed out waiting for %s", sock)
}
