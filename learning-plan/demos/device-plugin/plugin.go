package main

import (
	"context"
	"fmt"
	"net"
	"os"
	"sync"
	"time"

	"google.golang.org/grpc"
	"k8s.io/klog/v2"
	pluginapi "k8s.io/kubelet/pkg/apis/deviceplugin/v1beta1"
)

// FakePlugin 实现 pluginapi.DevicePluginServer (Device Plugin v1beta1)。
//
// 关键约定:
//   - ListAndWatch 是一个长连接 stream。kubelet 注册成功后会立刻调用,
//     插件先发一次全量 Device 列表, 之后只在设备增/删/健康变化时再 Send 新的全量。
//   - Allocate 在每个容器创建前被 kubelet 调用一次, 入参是 DeviceManager
//     已经从空闲池中挑出的 deviceIDs, 出参是要注入容器的 envs/mounts/devices。
//   - 同一进程内 ListAndWatch 必须一直存活, 它返回即视为插件不可用。
type FakePlugin struct {
	pluginapi.UnimplementedDevicePluginServer

	devices []*pluginapi.Device

	server *grpc.Server
	mu     sync.Mutex
	stopCh chan struct{}
}

func newFakePlugin(n int) *FakePlugin {
	var devs []*pluginapi.Device
	for i := 0; i < n; i++ {
		devs = append(devs, &pluginapi.Device{
			ID:     fmt.Sprintf("fake-device-%d", i),
			Health: pluginapi.Healthy,
		})
	}
	return &FakePlugin{devices: devs, stopCh: make(chan struct{})}
}

// serve 在 pluginSock 上启动 gRPC server, 并向其注册 DevicePluginServer。
func (p *FakePlugin) serve() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	// 删除旧 socket, 否则 net.Listen 会报 address already in use。
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
		klog.Infof("device plugin listening on %s", pluginSock)
		if err := p.server.Serve(lis); err != nil {
			klog.Errorf("grpc serve exited: %v", err)
		}
	}()

	// 等 server 真正起来再返回, 否则后续 Register 调用 kubelet 时,
	// kubelet 反向 dial 我们这边会失败。
	return waitForSocket(pluginSock, 5*time.Second)
}

// stop 优雅停止 gRPC server。
func (p *FakePlugin) stop() {
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

// GetDevicePluginOptions 由 kubelet 在 Register 后调用一次, 协商可选能力。
func (p *FakePlugin) GetDevicePluginOptions(_ context.Context, _ *pluginapi.Empty) (*pluginapi.DevicePluginOptions, error) {
	return &pluginapi.DevicePluginOptions{
		PreStartRequired:                false,
		GetPreferredAllocationAvailable: false,
	}, nil
}

// ListAndWatch 上报当前设备列表与健康状态。kubelet 据此更新
// Node.status.capacity["learning-plan.io/fake-device"]。
// 真实实现会用 NVML / 厂商 SDK 周期巡检设备, 不健康时从列表移除并 Send。
func (p *FakePlugin) ListAndWatch(_ *pluginapi.Empty, srv pluginapi.DevicePlugin_ListAndWatchServer) error {
	// 1) 立刻发一次全量。
	if err := srv.Send(&pluginapi.ListAndWatchResponse{Devices: p.devices}); err != nil {
		return err
	}
	klog.Infof("ListAndWatch: pushed %d devices", len(p.devices))

	// 2) 进入阻塞循环。Demo 里设备永远健康, 所以仅在 ticker 上「保活心跳」
	//    式重发一次列表 (实际 kubelet 不要求, 但便于演示)。退出条件:
	//    server 被停或 client 取消。
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

// Allocate 把 kubelet 选中的若干 deviceIDs 转换为「注入容器的合同」:
// 环境变量、挂载、设备节点。本 demo 用一个 hostPath 假目录 /tmp/fake-devices
// 模拟厂商 SDK 路径, 让 Pod 能拿到一个真实存在的挂载点验证链路。
func (p *FakePlugin) Allocate(_ context.Context, req *pluginapi.AllocateRequest) (*pluginapi.AllocateResponse, error) {
	resp := &pluginapi.AllocateResponse{}
	for _, creq := range req.ContainerRequests {
		klog.Infof("Allocate: container requests devices=%v", creq.DevicesIDs)
		car := &pluginapi.ContainerAllocateResponse{
			Envs: map[string]string{
				// 业务容器据此知道自己拿到了哪些设备。NVIDIA 插件用类似的
				// NVIDIA_VISIBLE_DEVICES 让 NVIDIA Container Toolkit 接管。
				"FAKE_VISIBLE_DEVICES": join(creq.DevicesIDs),
				"FAKE_RESOURCE_NAME":   resourceName,
			},
			Mounts: []*pluginapi.Mount{
				{
					ContainerPath: "/etc/fake-devices",
					HostPath:      "/tmp/fake-devices",
					ReadOnly:      true,
				},
			},
			// 不挂真实 /dev 节点 (容器可能没权限), 仅做示意:
			// 真实 GPU 插件会在这里放 /dev/nvidia0 / /dev/nvidiactl 等。
			Devices: []*pluginapi.DeviceSpec{},
			Annotations: map[string]string{
				"learning-plan.io/allocated-ids": join(creq.DevicesIDs),
			},
		}
		resp.ContainerResponses = append(resp.ContainerResponses, car)
	}
	return resp, nil
}

// PreStartContainer 由 kubelet 在容器启动前调用 (仅当 GetDevicePluginOptions
// 中 PreStartRequired=true)。本 demo 不需要, 直接返回空。
func (p *FakePlugin) PreStartContainer(_ context.Context, _ *pluginapi.PreStartContainerRequest) (*pluginapi.PreStartContainerResponse, error) {
	return &pluginapi.PreStartContainerResponse{}, nil
}

// GetPreferredAllocation 由 kubelet 在 DeviceManager 准备分配时调用 (可选)。
// 本 demo 关闭该能力 (GetPreferredAllocationAvailable=false), 留空兜底实现。
func (p *FakePlugin) GetPreferredAllocation(_ context.Context, _ *pluginapi.PreferredAllocationRequest) (*pluginapi.PreferredAllocationResponse, error) {
	return &pluginapi.PreferredAllocationResponse{}, nil
}

// waitForSocket 轮询直到指定 socket 可以被 dial, 避免 race。
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

func join(ids []string) string {
	out := ""
	for i, id := range ids {
		if i > 0 {
			out += ","
		}
		out += id
	}
	return out
}
