// fake-gpu: 一个最小可运行的 GPU 风格 Kubernetes Device Plugin。
//
// 它向 kubelet 宣告 4 个虚拟 GPU (fake-gpu.k8s.io/gpu),
// 设备 ID 采用 NVIDIA 风格的 UUID 字符串
// (GPU-00000000-0000-0000-0000-000000000000 .. -000000000003),
// 实现 ListAndWatch / Allocate / GetDevicePluginOptions / PreStartContainer
// 四个 gRPC 方法, 并在 kubelet.sock 被重建时自动重新注册.
//
// 与同目录 ../device-plugin demo 的区别:
//   - 资源名改为 fake-gpu.k8s.io/gpu, 数量改为 4.
//   - Allocate 返回 NVIDIA 风格的 envs (NVIDIA_VISIBLE_DEVICES, NVIDIA_DRIVER_CAPABILITIES),
//     演示"插件返回 env -> CRI CreateContainer 注入 -> container runtime 据此挂载设备"的真实模型.
//   - 不挂任何宿主机设备节点 (NVIDIA 也不这么做, 设备注入靠 nvidia-container-runtime 的 prestart hook).
//
// 部署: kubectl apply -f daemonset.yaml
// 查看: kubectl describe node | grep fake-gpu.k8s.io/gpu
// 消费: kubectl apply -f pod-consumer.yaml; kubectl logs fake-gpu-consumer
package main

import (
	"context"
	"net"
	"os"
	"os/signal"
	"path"
	"syscall"
	"time"

	"github.com/fsnotify/fsnotify"
	"google.golang.org/grpc"
	"k8s.io/klog/v2"
	pluginapi "k8s.io/kubelet/pkg/apis/deviceplugin/v1beta1"
)

const (
	// resourceName 必须是 DNS Label 风格的 Extended Resource 名.
	// 模仿 nvidia.com/gpu 的命名形式.
	resourceName = "fake-gpu.k8s.io/gpu"

	// kubelet 监听注册请求的固定 socket. DaemonSet 通过 hostPath
	// 把 /var/lib/kubelet/device-plugins 挂进容器, 因此插件能同时连这个 socket
	// 和创建自己的 socket.
	kubeletSock = pluginapi.DevicePluginPath + "kubelet.sock"

	// 插件自己的 socket 文件名 (basename, kubelet 会拼出绝对路径).
	pluginSockName = "fake-gpu.sock"
	pluginSock     = pluginapi.DevicePluginPath + pluginSockName

	// 模拟 4 个虚拟 GPU.
	numDevices = 4
)

func main() {
	klog.InitFlags(nil)
	defer klog.Flush()
	klog.Infof("starting fake-gpu device plugin, resource=%s, devices=%d", resourceName, numDevices)

	plugin := newFakeGPU(numDevices)

	// (1) 启动自身的 gRPC server.
	if err := plugin.serve(); err != nil {
		klog.Fatalf("serve: %v", err)
	}
	defer plugin.stop()

	// (2) 首次注册到 kubelet.
	if err := plugin.register(); err != nil {
		klog.Fatalf("register: %v", err)
	}
	klog.Info("registered with kubelet")

	// (3) 监听 kubelet.sock 重建事件; kubelet 重启会重建该文件,
	// 此时已注册的插件 endpoint 会被清空, 必须重新 Register.
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
			// kubelet.sock 被重建意味着 kubelet 刚重启过.
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

// register 拨号到 kubelet.sock, 调用 Registration.Register
// 把当前插件 (resource_name + endpoint socket) 挂进 kubelet 的 DeviceManager.
func (p *FakeGPU) register() error {
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
		Version:      pluginapi.Version,         // "v1beta1"
		Endpoint:     path.Base(pluginSockName), // kubelet 会拼出 pluginapi.DevicePluginPath + endpoint
		ResourceName: resourceName,              // fake-gpu.k8s.io/gpu
		Options: &pluginapi.DevicePluginOptions{
			PreStartRequired:                false,
			GetPreferredAllocationAvailable: false,
		},
	})
	return err
}
