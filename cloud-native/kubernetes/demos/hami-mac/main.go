// hami-mac: 一个在 Mac 上跑 kind 就能复现 HAMi 关键链路的 fake device plugin.
//
// 它把"1 张物理 GPU 上报为 N 个 vGPU 切片"和"Allocate 注入 HAMi 风格的
// LD_PRELOAD + CUDA_DEVICE_MEMORY_LIMIT_x + CUDA_DEVICE_SM_LIMIT_x env"这两件
// HAMi 最关键的事做出来, 让你在没有 NVIDIA GPU 的笔记本上也能看到完整的
// kubelet -> DeviceManager -> Allocate -> CRI -> 容器 env 链路.
//
// 与同目录 ../fake-gpu 的区别:
//   - fake-gpu 上报 1:1 的 4 张物理卡, 学的是 NVIDIA Device Plugin 的基本 SHAPE.
//   - 本 demo 上报 1:vgpuPerCard 的 N 张 vGPU 切片 (默认 4 phys × 10 = 40),
//     注入 LD_PRELOAD + CUDA_DEVICE_*_LIMIT_x, 学的是 HAMi 在 fake-gpu 基础上
//     多做的「切片上报 + 配额 env 注入」两件事.
//
// 故意不实现的部分:
//   - HAMi-webhook: 真实 HAMi 用 mutating webhook 把 Pod 申明的 nvidia.com/gpumem 注成 annotation.
//     本 demo 直接在 Allocate 里读取 ENV_HAMI_DEFAULT_MEM / CORES 环境变量当默认值.
//   - HAMi-scheduler-extender: 真实 HAMi 由 extender 决定具体哪块物理卡, 把 UUID 写回 annotation.
//     本 demo 由 kubelet DeviceManager 随便挑一个 vGPU slice ID 即可演示.
//   - libvgpu.so: 真实 HAMi 在容器内通过 LD_PRELOAD 加载 C 库 hook cuMemAlloc.
//     Mac 没有 NVIDIA driver, 我们仅在 env 里注入 LD_PRELOAD 路径让你看到 env 注入这一步通了,
//     真正的 hook 实现需要在阶段 6 上云租 GPU 才能跑.
//
// 部署: kubectl apply -f daemonset.yaml
// 查看: kubectl describe node | grep nvidia.com
// 消费: kubectl apply -f pod-hami-consumer.yaml; kubectl logs hami-consumer
package main

import (
	"context"
	"net"
	"os"
	"os/signal"
	"path"
	"strconv"
	"syscall"
	"time"

	"github.com/fsnotify/fsnotify"
	"google.golang.org/grpc"
	"k8s.io/klog/v2"
	pluginapi "k8s.io/kubelet/pkg/apis/deviceplugin/v1beta1"
)

const (
	resourceName = "nvidia.com/gpu"

	kubeletSock = pluginapi.DevicePluginPath + "kubelet.sock"

	pluginSockName = "hami-mac.sock"
	pluginSock     = pluginapi.DevicePluginPath + pluginSockName

	// 默认 4 张物理卡, 每张切 10 份 vGPU. 与 HAMi 默认配置一致.
	defaultPhysCards   = 4
	defaultVGPUPerCard = 10

	// 容器内 libvgpu.so 的路径. 真实 HAMi 由 daemonset hostPath 把宿主机
	// /usr/local/vgpu/ 挂进消费 Pod (通过 webhook 注入 volumes/mounts).
	// 本 demo 仅注入 env, 容器里不会真的 dlopen 这个文件.
	libvgpuPath = "/usr/local/vgpu/libvgpu.so"
)

func envInt(key string, def int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return def
}

func main() {
	klog.InitFlags(nil)
	defer klog.Flush()

	phys := envInt("HAMI_MAC_PHYS_CARDS", defaultPhysCards)
	per := envInt("HAMI_MAC_VGPU_PER_CARD", defaultVGPUPerCard)
	defMem := envInt("HAMI_MAC_DEFAULT_MEM", 3000)
	defCores := envInt("HAMI_MAC_DEFAULT_CORES", 30)

	klog.Infof("starting hami-mac fake plugin: resource=%s phys=%d perCard=%d -> %d vGPU slices",
		resourceName, phys, per, phys*per)
	klog.Infof("Allocate defaults: gpumem=%dMiB cores=%d%%", defMem, defCores)

	plugin := newHAMiFake(phys, per, defMem, defCores)

	if err := plugin.serve(); err != nil {
		klog.Fatalf("serve: %v", err)
	}
	defer plugin.stop()

	if err := plugin.register(); err != nil {
		klog.Fatalf("register: %v", err)
	}
	klog.Info("registered with kubelet")

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		klog.Fatalf("fsnotify: %v", err)
	}
	defer watcher.Close()
	if err := watcher.Add(pluginapi.DevicePluginPath); err != nil {
		klog.Fatalf("watch %s: %v", pluginapi.DevicePluginPath, err)
	}

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)

	for {
		select {
		case ev := <-watcher.Events:
			if ev.Name == kubeletSock && (ev.Op&fsnotify.Create) == fsnotify.Create {
				klog.Warning("kubelet.sock recreated, re-registering")
				plugin.stop()
				if err := plugin.serve(); err != nil {
					klog.Errorf("re-serve: %v", err)
					continue
				}
				time.Sleep(time.Second)
				if err := plugin.register(); err != nil {
					klog.Errorf("re-register: %v", err)
					continue
				}
				klog.Info("re-registered with kubelet")
			}
		case err := <-watcher.Errors:
			klog.Errorf("fsnotify error: %v", err)
		case <-stop:
			klog.Info("shutting down")
			return
		}
	}
}

func (p *HAMiFake) register() error {
	conn, err := grpc.Dial(
		"unix://"+kubeletSock,
		grpc.WithInsecure(),
		grpc.WithBlock(),
		grpc.WithTimeout(5*time.Second),
		grpc.WithContextDialer(func(ctx context.Context, addr string) (net.Conn, error) {
			d := &net.Dialer{Timeout: 5 * time.Second}
			return d.DialContext(ctx, "unix", addr[len("unix://"):])
		}),
	)
	if err != nil {
		return err
	}
	defer conn.Close()
	client := pluginapi.NewRegistrationClient(conn)
	_, err = client.Register(context.Background(), &pluginapi.RegisterRequest{
		Version:      pluginapi.Version,
		Endpoint:     path.Base(pluginSockName),
		ResourceName: resourceName,
		Options: &pluginapi.DevicePluginOptions{
			PreStartRequired:                false,
			GetPreferredAllocationAvailable: false,
		},
	})
	return err
}
