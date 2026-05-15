// fake-device-plugin: 一个最小可运行的 Kubernetes Device Plugin demo。
//
// 它向 kubelet 宣告 8 个虚拟设备 (learning-plan.io/fake-device-0 .. -7),
// 实现 ListAndWatch / Allocate / GetDevicePluginOptions / PreStartContainer
// 四个 gRPC 方法, 并在 kubelet.sock 被重建时自动重新注册。
//
// 部署: kubectl apply -f daemonset.yaml
// 查看: kubectl describe node <node> | grep learning-plan.io/fake-device
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
	// resourceName 必须是 DNS Label 风格的 Extended Resource 名。
	// kubelet 会用它作为 Node.status.capacity 的 key。
	resourceName = "learning-plan.io/fake-device"

	// kubelet 监听注册请求的固定 socket。Device Plugin DaemonSet 通过 hostPath
	// 把 /var/lib/kubelet/device-plugins 挂进容器, 因此插件能同时连这个 socket
	// 和创建自己的 socket。
	kubeletSock = pluginapi.DevicePluginPath + "kubelet.sock"

	// 插件自己的 socket 文件名 (注意是 basename, kubelet 会自动拼前缀路径)。
	pluginSockName = "learning-plan-fake-device.sock"
	pluginSock     = pluginapi.DevicePluginPath + pluginSockName

	// 模拟 8 个虚拟设备。
	numDevices = 8
)

func main() {
	klog.InitFlags(nil)
	defer klog.Flush()
	klog.Infof("starting fake device plugin, resource=%s, devices=%d", resourceName, numDevices)

	plugin := newFakePlugin(numDevices)

	// (1) 启动自身的 gRPC server。
	if err := plugin.serve(); err != nil {
		klog.Fatalf("serve: %v", err)
	}
	defer plugin.stop()

	// (2) 首次注册到 kubelet。
	if err := plugin.register(); err != nil {
		klog.Fatalf("register: %v", err)
	}
	klog.Info("registered with kubelet")

	// (3) 监听 kubelet.sock 重建事件; kubelet 重启会重建该文件,
	// 此时已注册的插件 endpoint 会被清空, 必须重新 Register。
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
			// kubelet.sock 被重建意味着 kubelet 刚重启过。
			if ev.Name == kubeletSock && (ev.Op&fsnotify.Create) == fsnotify.Create {
				klog.Warning("kubelet.sock recreated, re-registering")
				// 重启自己的 server, 然后重新注册。
				plugin.stop()
				if err := plugin.serve(); err != nil {
					klog.Errorf("re-serve: %v", err)
					continue
				}
				// kubelet 启动后需要一点时间才能接受 Register, 简单 sleep 即可。
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
// 把当前插件 (resource_name + endpoint socket) 挂进 kubelet 的 DeviceManager。
func (p *FakePlugin) register() error {
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
		Version:      pluginapi.Version,           // "v1beta1"
		Endpoint:     path.Base(pluginSockName),   // kubelet 会拼出 pluginapi.DevicePluginPath + endpoint
		ResourceName: resourceName,                // Extended Resource 名
		Options: &pluginapi.DevicePluginOptions{
			PreStartRequired:                false,
			GetPreferredAllocationAvailable: false,
		},
	})
	return err
}
