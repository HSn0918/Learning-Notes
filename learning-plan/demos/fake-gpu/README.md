# fake-gpu

一个最小可运行的 GPU 风格 Kubernetes Device Plugin demo。它在不需要真实 GPU 硬件的前提下，模仿 NVIDIA k8s-device-plugin 的关键 SHAPE：4 块 fake GPU、UUID 风格 ID、`Allocate` 返回 NVIDIA 风格的 envs。让你在 Mac 上用 kind 就能跑通整条链路。

配套阅读：
- [[../../gpu-scheduling-source]]（源码导读）
- [[../../../cloud-native/kubernetes/control-plane/gpu-scheduling]]（概念层）
- [[demo-fake-gpu]]（本目录的笔记入口）

## 文件结构

| 文件 | 作用 |
| :--- | :--- |
| `main.go` | 进程入口：启动 gRPC server、首次 Register、用 fsnotify 监听 kubelet.sock 重建并重新注册 |
| `plugin.go` | `FakeGPU` 结构体，实现 `pluginapi.DevicePluginServer` 的 4 个 RPC，Allocate 返回 `NVIDIA_VISIBLE_DEVICES` env |
| `daemonset.yaml` | 把 `/var/lib/kubelet/device-plugins/` 挂进容器的 DaemonSet |
| `pod-consumer.yaml` | 请 1 块 `fake-gpu.k8s.io/gpu` 的 busybox Pod，启动后打印注入的 env |
| `go.mod` | 依赖 `k8s.io/kubelet`（pluginapi）、`google.golang.org/grpc`、`fsnotify` |
| `demo-fake-gpu.md` | 笔记入口（含 walkthrough）|

## 与 `../device-plugin` 的区别

如果你已经看过 `learning-plan/demos/device-plugin` 那个通用 fake device 示例，这里有三个有意识的差异点：

1. **资源名** `learning-plan.io/fake-device` → `fake-gpu.k8s.io/gpu`，对齐 NVIDIA `nvidia.com/gpu` 的形态。
2. **设备 ID** 用 UUID 字符串 `GPU-00000000-0000-0000-0000-00000000000{0..3}`，让链路日志看起来像真的 NVIDIA 卡。
3. **Allocate 不返回 Mounts / Devices**，只返回 `NVIDIA_VISIBLE_DEVICES` env，演示「环境变量注入 → container runtime 据此动态挂载设备」这条 NVIDIA 模式——而不是把 `/dev/nvidiaX` 写死在插件里。这是真实 NVIDIA plugin 的默认行为，详见 [[../../gpu-scheduling-source]] 的「为什么 Allocate 不直接 mount /dev/nvidiaX」一节。

## 本地构建

```bash
cd learning-plan/demos/fake-gpu
go mod tidy
go build ./...
```

把镜像打出来（同目录写一个简单 Dockerfile，可复用 `../device-plugin/Dockerfile` 的多阶段构建思路）：

```bash
docker build -t learning-notes/fake-gpu:latest -f ../device-plugin/Dockerfile .
# 或者你自己写一个 Dockerfile 见下
```

最简 Dockerfile：

```Dockerfile
FROM golang:1.26 AS builder
WORKDIR /workspace
COPY go.mod ./
RUN go mod download || true
COPY *.go ./
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags='-s -w' -o /out/fake-gpu .

FROM gcr.io/distroless/static:nonroot
COPY --from=builder /out/fake-gpu /fake-gpu
USER 0:0
ENTRYPOINT ["/fake-gpu"]
```

## 在 kind 集群上验证（Mac 也能跑）

```bash
# 1) 起一个单节点 kind 集群
kind create cluster --name fake-gpu

# 2) 把镜像 load 进 kind 节点
kind load docker-image learning-notes/fake-gpu:latest --name fake-gpu

# 3) 部署 DaemonSet
kubectl apply -f daemonset.yaml

# 4) 等 plugin Pod Ready
kubectl -n kube-system get pod -l app=fake-gpu-plugin -w
# 看到 Running 后 Ctrl+C

# 5) 验证 Extended Resource 已经出现在 Node capacity 中
kubectl describe node | grep fake-gpu.k8s.io/gpu
# 期望:
#   fake-gpu.k8s.io/gpu: 4    (在 Capacity / Allocatable 两段都能看到)
```

## 请一个 Pod 消费这个资源

```bash
kubectl apply -f pod-consumer.yaml

# 等 Pod 跑起来
kubectl get pod fake-gpu-consumer -w

# 看注入的 env
kubectl logs fake-gpu-consumer
```

期望输出：

```
=== 来自 fake-gpu device plugin 的 envs ===
FAKE_GPU_DEVICES=GPU-00000000-0000-0000-0000-000000000000
FAKE_RESOURCE_NAME=fake-gpu.k8s.io/gpu
NVIDIA_DRIVER_CAPABILITIES=compute,utility
NVIDIA_VISIBLE_DEVICES=GPU-00000000-0000-0000-0000-000000000000

=== 真实 NVIDIA 环境会在这里 mount /dev/nvidiaX, 本 demo 没有真实 runtime, 故 /dev 中无 GPU 设备:
(空, 符合预期)
```

如果你看到 `NVIDIA_VISIBLE_DEVICES=GPU-00000000-...`，整条链路就跑通了：

```
plugin ListAndWatch -> kubelet 写 Node.capacity 4 张 fake GPU
Scheduler NodeResourcesFit -> 把 Pod bind 到这台 Node
kubelet DeviceManager.Allocate -> 挑 1 个 UUID, 调插件 Allocate
plugin -> 返回 envs{NVIDIA_VISIBLE_DEVICES=GPU-xxx}
kubelet kuberuntime -> 把 envs 合并进 CRI CreateContainer
containerd -> 启动容器 (env 注入完成)
容器进程 -> env 中看到 NVIDIA_VISIBLE_DEVICES
```

差最后一步：在真实 GPU 节点上，`nvidia-container-runtime` 的 prestart hook 会读 `NVIDIA_VISIBLE_DEVICES`，把 `/dev/nvidia0` 与 driver 库 bind-mount 进容器，于是容器里 `nvidia-smi` 能输出。

## 清理

```bash
kubectl delete pod fake-gpu-consumer
kubectl delete -f daemonset.yaml
kind delete cluster --name fake-gpu
```

## 常见问题

- **Pod 一直 Pending，事件提示 `Insufficient fake-gpu.k8s.io/gpu`**：
  插件 Pod 还没把 `ListAndWatch` 数据发给 kubelet。`kubectl -n kube-system logs <plugin-pod>` 看是否出现 `registered with kubelet` 和 `ListAndWatch: pushed 4 fake GPUs`。
- **`kubectl describe node` 看不到 `fake-gpu.k8s.io/gpu`**：
  通常是 DaemonSet 没启动或 hostPath 没挂上；进入插件 Pod `ls /var/lib/kubelet/device-plugins/` 看是否能看到 `kubelet.sock`。
- **`kubelet 重启后 fake-gpu.k8s.io/gpu 数量变 0`**：
  fsnotify 没正常工作。看插件日志是否打印 `kubelet.sock recreated, re-registering`。本 demo 监听的是 `/var/lib/kubelet/device-plugins/` 目录而不是单个文件，这是正确做法。
- **busybox Pod 看不到 `NVIDIA_VISIBLE_DEVICES`**：
  说明 `Allocate` 链路没生效。检查插件日志是否打印 `Allocate: container requests GPUs=[GPU-...]`；若打印了但容器里看不到 env，说明 kubelet 和 CRI 之间合并环节坏了（实际几乎不会发生）。
