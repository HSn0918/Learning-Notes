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

// HAMiFake 是一个学 HAMi 的 fake device plugin.
//
// 与 ../fake-gpu 的两点关键不同:
//
//  1. ListAndWatch 上报的不是 N 张物理卡, 而是 phys × vgpuPerCard 个 vGPU 切片.
//     设备 ID 形如 GPU-...-phys-3-vgpu-7, 同一物理卡的 10 个切片共享同一 phys 段,
//     这样真实 HAMi-scheduler 在 Filter 时就能按 prefix 把同卡的切片聚合.
//
//  2. Allocate 多注入了三类 env:
//        LD_PRELOAD                    -> 容器启动后加载 libvgpu.so
//        CUDA_DEVICE_MEMORY_LIMIT_X    -> libvgpu hook cuMemAlloc 时的上限
//        CUDA_DEVICE_SM_LIMIT_X        -> libvgpu hook cuLaunchKernel 时的上限
//     这是 HAMi 实现配额的"入口契约": kubelet 注入 env, libvgpu 在容器内读 env.
type HAMiFake struct {
	pluginapi.UnimplementedDevicePluginServer

	devices  []*pluginapi.Device
	defMem   int // MiB, 当 Pod 没显式声明 nvidia.com/gpumem 时的默认值
	defCores int // %

	server *grpc.Server
	mu     sync.Mutex
	stopCh chan struct{}
}

func newHAMiFake(phys, perCard, defMem, defCores int) *HAMiFake {
	var devs []*pluginapi.Device
	for p := 0; p < phys; p++ {
		// 用同一段 phys-UUID 让真实 HAMi-scheduler 能按物理卡聚合 vGPU.
		// 这里仍然用全零方便复现.
		physUUID := fmt.Sprintf("GPU-00000000-0000-0000-0000-00000000000%d", p)
		for i := 0; i < perCard; i++ {
			devs = append(devs, &pluginapi.Device{
				ID:     fmt.Sprintf("%s-vgpu-%d", physUUID, i),
				Health: pluginapi.Healthy,
			})
		}
	}
	return &HAMiFake{
		devices:  devs,
		defMem:   defMem,
		defCores: defCores,
		stopCh:   make(chan struct{}),
	}
}

func (p *HAMiFake) serve() error {
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
		klog.Infof("hami-mac plugin listening on %s", pluginSock)
		if err := p.server.Serve(lis); err != nil {
			klog.Errorf("grpc serve exited: %v", err)
		}
	}()

	return waitForSocket(pluginSock, 5*time.Second)
}

func (p *HAMiFake) stop() {
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

func (p *HAMiFake) GetDevicePluginOptions(_ context.Context, _ *pluginapi.Empty) (*pluginapi.DevicePluginOptions, error) {
	return &pluginapi.DevicePluginOptions{
		PreStartRequired:                false,
		GetPreferredAllocationAvailable: false,
	}, nil
}

// ListAndWatch: 把 phys × perCard 个切片当独立 device 上报给 kubelet.
// 因此 Node.status.capacity["nvidia.com/gpu"] 会是 phys * perCard, 例如 40.
// 真实 HAMi 同样这么做 —— 显存 / 算力的限制不在数量上, 由 Allocate env + libvgpu 协作完成.
func (p *HAMiFake) ListAndWatch(_ *pluginapi.Empty, srv pluginapi.DevicePlugin_ListAndWatchServer) error {
	if err := srv.Send(&pluginapi.ListAndWatchResponse{Devices: p.devices}); err != nil {
		return err
	}
	klog.Infof("ListAndWatch: pushed %d vGPU slices", len(p.devices))

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

// Allocate 模仿 HAMi 注入"配额契约 env".
//
// 真实 HAMi 的字段来源:
//   - Pod annotation hami.io/vgpu-devices-to-allocate (由 HAMi-scheduler 写)
//   - Pod resources.limits["nvidia.com/gpumem"] (由用户声明)
//
// 本 demo 简化为: defMem / defCores 由进程环境变量决定, 不区分容器.
// 目的是让你在 Mac 上看到 env 注入的"形状", 不追求和真实 HAMi 一比一对齐.
func (p *HAMiFake) Allocate(_ context.Context, req *pluginapi.AllocateRequest) (*pluginapi.AllocateResponse, error) {
	resp := &pluginapi.AllocateResponse{}
	for _, creq := range req.ContainerRequests {
		ids := creq.DevicesIDs
		klog.Infof("Allocate: container requests vGPU slices=%v", ids)

		envs := map[string]string{
			// 真实 NVIDIA runtime 仍然读这个, 决定挂哪些 /dev/nvidiaX.
			// HAMi 把切片 ID 的 phys-UUID 段提取出来作为这里的值.
			"NVIDIA_VISIBLE_DEVICES":     joinPhysUUIDs(ids),
			"NVIDIA_DRIVER_CAPABILITIES": "compute,utility",

			// HAMi 注入的 LD_PRELOAD: 容器启动时加载 libvgpu.so,
			// hook libcuda.so 的 cuMemAlloc / cuLaunchKernel 等 API.
			// Mac 上没有 libcuda.so, 容器里这个 LD_PRELOAD 路径文件不存在,
			// dlopen 会失败但不影响 demo —— 我们只是验证 env 注入路径通了.
			"LD_PRELOAD": libvgpuPath,
		}

		// HAMi 对每张切片注入独立的 CUDA_DEVICE_MEMORY_LIMIT_{i} / CUDA_DEVICE_SM_LIMIT_{i}.
		// i 是容器内 CUDA visible device index, 从 0 开始数.
		for i := range ids {
			envs[fmt.Sprintf("CUDA_DEVICE_MEMORY_LIMIT_%d", i)] = fmt.Sprintf("%dm", p.defMem)
			envs[fmt.Sprintf("CUDA_DEVICE_SM_LIMIT_%d", i)] = fmt.Sprintf("%d", p.defCores)
		}

		envs["HAMI_FAKE_VGPU_IDS"] = strings.Join(ids, ",")

		car := &pluginapi.ContainerAllocateResponse{
			Envs: envs,
			Annotations: map[string]string{
				"hami.io/vgpu-devices-allocated": strings.Join(ids, ","),
			},
			// 真实 HAMi 会通过 webhook 注入 volumes/mounts 把宿主机
			// /usr/local/vgpu/ 挂到容器 /usr/local/vgpu/.
			// 本 demo 不模拟 webhook, 容器里 libvgpuPath 这个文件不存在 (OK).
		}
		resp.ContainerResponses = append(resp.ContainerResponses, car)
	}
	return resp, nil
}

func (p *HAMiFake) PreStartContainer(_ context.Context, _ *pluginapi.PreStartContainerRequest) (*pluginapi.PreStartContainerResponse, error) {
	return &pluginapi.PreStartContainerResponse{}, nil
}

func (p *HAMiFake) GetPreferredAllocation(_ context.Context, _ *pluginapi.PreferredAllocationRequest) (*pluginapi.PreferredAllocationResponse, error) {
	return &pluginapi.PreferredAllocationResponse{}, nil
}

// joinPhysUUIDs 把 vGPU 切片 ID "GPU-...-vgpu-7" 还原成物理卡 UUID "GPU-...",
// 然后去重 + 用逗号拼起来 —— 因为 NVIDIA_VISIBLE_DEVICES 应该是物理 UUID 集合.
func joinPhysUUIDs(ids []string) string {
	seen := map[string]struct{}{}
	var out []string
	for _, id := range ids {
		// "GPU-00000000-0000-0000-0000-000000000000-vgpu-7"
		if idx := strings.Index(id, "-vgpu-"); idx > 0 {
			id = id[:idx]
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	return strings.Join(out, ",")
}

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
