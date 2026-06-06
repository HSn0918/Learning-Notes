# fake-device-plugin

> **状态**：✅ Mac 编过（先 `go mod tidy`）；需 kind/真集群部署 · 详见 [demos 验证总表](../README.md)

一个最小可运行的 Kubernetes Device Plugin demo，向 kubelet 宣告 8 个名为
`learning-plan.io/fake-device` 的虚拟设备。配套阅读：
[[kubelet-cri-source]] | [[gpu-scheduling]]

## 文件结构

| 文件 | 作用 |
| :--- | :--- |
| `main.go` | 进程入口：启动 gRPC server、首次 Register、用 fsnotify 监听 kubelet.sock 重建并重新注册 |
| `plugin.go` | `FakePlugin` 结构体，实现 `pluginapi.DevicePluginServer` 的 4 个 RPC |
| `Dockerfile` | 多阶段构建静态二进制 + distroless 运行镜像 |
| `daemonset.yaml` | 把 `/var/lib/kubelet/device-plugins/` 挂进容器的 DaemonSet |
| `go.mod` | 依赖 `k8s.io/kubelet`（pluginapi）、`google.golang.org/grpc`、`fsnotify` |
| `demo-device-plugin.md` | 笔记入口（含 walkthrough）|

## 本地构建

```bash
go mod tidy
go build ./...
# 或直接出镜像
docker build -t learning-notes/fake-device-plugin:latest .
```

## 在 kind 集群上验证

```bash
# 1) 起一个单节点 kind 集群
kind create cluster --name dp

# 2) 把镜像 load 进 kind 节点
kind load docker-image learning-notes/fake-device-plugin:latest --name dp

# 3) 部署 DaemonSet
kubectl apply -f daemonset.yaml

# 4) 等 Pod Ready
kubectl -n kube-system get pod -l app=fake-device-plugin -w

# 5) 验证 Extended Resource 已经出现在 Node capacity 中
kubectl describe node | grep learning-plan.io/fake-device
# 期望输出:
#   learning-plan.io/fake-device: 8
```

应该能在 `Capacity` 和 `Allocatable` 两段中各看到一行 `learning-plan.io/fake-device: 8`，
说明 kubelet 已经把 `ListAndWatch` 上报的 8 个设备写入了 Node 状态。

## 请一个 Pod 消费这个资源

```bash
cat <<EOF | kubectl apply -f -
apiVersion: v1
kind: Pod
metadata:
  name: fake-device-consumer
spec:
  restartPolicy: Never
  containers:
    - name: app
      image: busybox:1.36
      command: ["sh", "-c", "env | grep FAKE_; ls -la /etc/fake-devices; sleep 3600"]
      resources:
        limits:
          # Extended Resource 必须 requests == limits, 这是 K8s 的硬性约束
          learning-plan.io/fake-device: 2
EOF

# 看 Allocate 返回的 envs / mounts 是否真的注入了
kubectl logs fake-device-consumer
# 期望看到:
#   FAKE_VISIBLE_DEVICES=fake-device-0,fake-device-1
#   FAKE_RESOURCE_NAME=learning-plan.io/fake-device
#   /etc/fake-devices/  (目录存在)
```

同时 kubelet 节点上的插件 Pod 日志会打印 `Allocate: container requests devices=[fake-device-0 fake-device-1]`，
证明握手完整跑通：

```
kubelet 注册入口  --Register-->  插件
插件             --ListAndWatch->  kubelet  -> Node.status.capacity = 8
Scheduler        --bind 2-->      kubelet
kubelet          --Allocate-->    插件     -> envs+mounts 注入 CRI CreateContainer
```

## 清理

```bash
kubectl delete pod fake-device-consumer
kubectl delete -f daemonset.yaml
kind delete cluster --name dp
```

## 常见问题

- **Pod 一直 Pending，事件提示 `Insufficient learning-plan.io/fake-device`**：
  说明 DeviceManager 还没拿到 `ListAndWatch` 数据。检查插件 Pod 是否 Running、
  `kubectl logs <plugin-pod>` 看是否出现 `registered with kubelet`。
- **插件 Pod CrashLoopBackOff，日志 `dial unix .../kubelet.sock: connection refused`**：
  kubelet 还没把 socket 创建出来。生产实现会用退避重试包住 `register()`；本 demo 简单 panic。
- **kubelet 重启后插件不再上报**：fsnotify 必须监听 *目录* 而不是单个文件。本 demo 用
  `watcher.Add(pluginapi.DevicePluginPath)` 是正确做法。
