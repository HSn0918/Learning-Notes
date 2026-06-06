# hami-mac

> **状态**：✅ Mac 编过；需 docker build + kind load 部署 · 详见 [demos 验证总表](../README.md)

在 Mac（无 NVIDIA GPU）+ kind 集群上，复现 HAMi 关键链路的可运行 fake device plugin。一句话定位：[[demo-fake-gpu]] 上报的是 1:1 物理卡 + 仅 `NVIDIA_VISIBLE_DEVICES` env，这个 demo 升级为 **1 卡切 N 份 vGPU + 注入 `LD_PRELOAD` + `CUDA_DEVICE_MEMORY_LIMIT_*` + `CUDA_DEVICE_SM_LIMIT_*`**，对应 HAMi device-plugin 的两件最关键的事。

配套阅读：
- [[hami-learning-path]] 「Mac 无 GPU 实操方案」节
- [[demo-fake-gpu]] 前置 demo

## 它做了什么

| HAMi 实做 | 本 demo |
| :--- | :--- |
| `ListAndWatch` 把 1 张物理卡上报 N 份（默认 10）| ✅ `HAMI_MAC_PHYS_CARDS × HAMI_MAC_VGPU_PER_CARD` 控制 |
| `Allocate` 注入 `LD_PRELOAD=/usr/local/vgpu/libvgpu.so` | ✅ env 注入；但容器里这个文件不存在（OK，验证 env 路径） |
| `Allocate` 注入 `CUDA_DEVICE_MEMORY_LIMIT_X` / `CUDA_DEVICE_SM_LIMIT_X` | ✅ 按 vGPU 切片数注入 X=0..N-1 |
| HAMi-webhook 改写 Pod resources / 注入 volumeMount | ❌ 不做（在路径笔记里说明真实做法） |
| HAMi-scheduler-extender Filter/Bind 选 vGPU | ❌ 不做（kubelet DeviceManager 随便挑一个） |
| libvgpu.so 真正 hook `cuMemAlloc` / `cuLaunchKernel` | ❌ Mac 没 NVIDIA driver；阶段 6 上云租 GPU 才能做 |

**Mac 上能完整看到链路的部分**：plugin Register → ListAndWatch → kubelet 写 Node capacity → Scheduler 调度 → kubelet 调 Allocate → 容器内 env 注入 7 个变量（其中 `LD_PRELOAD` 和 `CUDA_DEVICE_*_LIMIT_*` 就是 HAMi 多出来的"配额契约"）。

## 本地构建

```bash
cd learning-plan/demos/hami-mac
go mod tidy
go build ./...
docker build -t learning-notes/hami-mac:latest .
```

## 在 kind 集群验证

```bash
# 1) 起 kind 集群（如果已经为 fake-gpu 起过，可以复用）
kind create cluster --name hami-mac

# 2) 把镜像 load 进 kind 节点
kind load docker-image learning-notes/hami-mac:latest --name hami-mac

# 3) 部署 plugin DaemonSet
kubectl apply -f daemonset.yaml
kubectl -n kube-system get pod -l app=hami-mac-plugin -w   # Running 后 Ctrl+C

# 4) 验证 Node capacity = 40 (4 × 10)
kubectl describe node | grep nvidia.com/gpu
# 期望:
#   nvidia.com/gpu: 40

# 5) apply 两个消费 Pod, 验证 env 注入
kubectl apply -f pod-hami-consumer.yaml
kubectl wait --for=condition=Ready pod/hami-consumer pod/hami-consumer-b --timeout=60s
kubectl logs hami-consumer
kubectl logs hami-consumer-b
```

期望输出（consumer-a）：

```
=== HAMi 风格的 envs (kubelet Allocate 注入) ===
CUDA_DEVICE_MEMORY_LIMIT_0=3000m
CUDA_DEVICE_SM_LIMIT_0=30
HAMI_FAKE_VGPU_IDS=GPU-00000000-0000-0000-0000-000000000000-vgpu-0
LD_PRELOAD=/usr/local/vgpu/libvgpu.so
NVIDIA_DRIVER_CAPABILITIES=compute,utility
NVIDIA_VISIBLE_DEVICES=GPU-00000000-0000-0000-0000-000000000000
```

看到 `LD_PRELOAD` + `CUDA_DEVICE_MEMORY_LIMIT_0=3000m` + `CUDA_DEVICE_SM_LIMIT_0=30`，HAMi device-plugin 这一块的"对外契约"就在你眼前了 —— 接下来差的就是容器里加载 libvgpu.so 真去 hook CUDA API。

## walkthrough

```
1) DaemonSet 起来 -> serve() -> register() -> kubelet 反向 dial
2) kubelet -> 我们: ListAndWatch()
   我们 -> kubelet: 40 个 vGPU 切片
        GPU-...000-vgpu-0 .. GPU-...000-vgpu-9     <- phys card 0 的 10 份
        GPU-...001-vgpu-0 .. GPU-...001-vgpu-9     <- phys card 1 的 10 份
        ...
   kubelet Node.status.capacity["nvidia.com/gpu"] = 40

3) kubectl apply pod-hami-consumer.yaml
   Scheduler NodeResourcesFit -> 选中 Node (capacity 40 够)
   Scheduler Bind -> Pod.spec.nodeName = <node>

4) kubelet SyncLoop -> DeviceManager.Allocate(pod, ctr)
        healthyDevices - allocatedDevices = 40 个空闲池
        挑出 ["GPU-...000-vgpu-3"]
        rpc 我们: Allocate({DevicesIDs:["GPU-...000-vgpu-3"]})
   我们 -> kubelet: ContainerAllocateResponse{
        Envs: {
            NVIDIA_VISIBLE_DEVICES:     "GPU-00000000-0000-0000-0000-000000000000",  // phys UUID
            NVIDIA_DRIVER_CAPABILITIES: "compute,utility",
            LD_PRELOAD:                 "/usr/local/vgpu/libvgpu.so",
            CUDA_DEVICE_MEMORY_LIMIT_0: "3000m",
            CUDA_DEVICE_SM_LIMIT_0:     "30",
            HAMI_FAKE_VGPU_IDS:         "GPU-...-vgpu-3",
        },
        Annotations: { "hami.io/vgpu-devices-allocated": "GPU-...-vgpu-3" },
   }

5) kuberuntime CreateContainer(envs=[...])
   -> 真实 HAMi 节点上: container 启动后 ld.so 先加载 LD_PRELOAD 指向的
                       libvgpu.so, 之后 hook libcuda.so 的 cuMemAlloc 等 API.
   -> 本 demo: dlopen /usr/local/vgpu/libvgpu.so 失败 (文件不存在),
              但 env 已经注入容器, kubectl logs 可见.
```

## 与真实 HAMi 的差距清单

| 真实 HAMi | 本 demo |
| :--- | :--- |
| webhook 改写 Pod spec、注入 volumeMounts | 不做（用进程 env 当默认值） |
| scheduler-extender 选 vGPU + 写 annotation | 不做（DeviceManager 随便挑） |
| ListAndWatch 用 NVML 真实读卡 UUID | 全零硬编码 UUID |
| Topology / NUMA 上报 | 不做 |
| 健康巡检（XID/ECC） | 永远 Healthy |
| libvgpu.so 真 hook CUDA API | 仅注入 env，无真实 hook |
| 多容器共享一卡的共享内存配额协商 | 无 |
| leader election + annotation reconcile | 无 |

## 进阶玩法

1. **跑真 HAMi-scheduler + 本 demo**：把真实 HAMi 的 scheduler/webhook chart 装到 kind 集群上，让它以为后端是 NVIDIA GPU。`webhook` 改写后的 Pod resources 走到 kubelet，kubelet 调本 demo 的 `Allocate`，env 链路完整。需要把 plugin 上报的资源名改成 HAMi-scheduler 认识的 `nvidia.com/vgpu`（视 HAMi 版本）。
2. **加 webhook 模拟**：自己写一个简化的 mutating webhook（[[controller-runtime-source]] 第 8 节做过），在 Pod 创建时把 `nvidia.com/gpumem` 解析成 annotation。
3. **kwok 大规模测试**：用 kwok 起 100 个虚拟节点，每个节点贴 `nvidia.com/gpu: 40` capacity label，让真 HAMi-scheduler 跑 Filter/Bind 看 spread vs binpack 策略效果（不能跑 Allocate，因为 kwok 没真 kubelet）。

## 清理

```bash
kubectl delete -f pod-hami-consumer.yaml
kubectl delete -f daemonset.yaml
kind delete cluster --name hami-mac
```

## 常见问题

- **Pod Pending 提示 `Insufficient nvidia.com/gpu`**：plugin 还没上报。看 plugin 日志是否有 `ListAndWatch: pushed 40 vGPU slices`。
- **`describe node` 看不到 `nvidia.com/gpu`**：DaemonSet 的 hostPath 没挂上；进 plugin Pod `ls /var/lib/kubelet/device-plugins/` 看 `kubelet.sock`。
- **容器里看不到 `CUDA_DEVICE_MEMORY_LIMIT_0`**：检查 plugin 日志是否打印 `Allocate: container requests vGPU slices=...`；若日志有但容器里没 env，说明 kubelet 这次 Allocate 没走到（极少见，通常是资源名拼错）。
- **`LD_PRELOAD` 让其它命令报 `error while loading shared libraries`**：因为容器里 `/usr/local/vgpu/libvgpu.so` 不存在。这正是 demo 的局限 —— 它只演示 env 注入。如果你介意 sh 在每次 exec 时报错，把 `LD_PRELOAD` 的值改成空字符串或 demo 镜像里真的放一个空 `.so`。
