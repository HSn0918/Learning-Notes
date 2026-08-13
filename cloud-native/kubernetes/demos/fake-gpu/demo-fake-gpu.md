#kubernetes #gpu #device-plugin #demo

相关笔记：[[gpu-scheduling-source]] | [[gpu-scheduling]] | [[k8s-development-roadmap]]

## 概述

本 demo 在 [[gpu-scheduling-source]] 的「手写简化复现」节落地为一个可运行项目。它实现一个 GPU 风格的虚拟设备插件，向 kubelet 宣告 4 个名为 `fake-gpu.k8s.io/gpu` 的设备，让我们在 Mac + kind（没有真实 GPU 硬件）的前提下走通整条 GPU 调度链路：插件 Register、`ListAndWatch` 上报 → kubelet 写入 Node `status.capacity` → Scheduler NodeResourcesFit 把 Pod 调度上来 → kubelet 调插件 `Allocate` → 返回的 **NVIDIA 风格 envs**（`NVIDIA_VISIBLE_DEVICES`）合并进 CRI `CreateContainer` → 容器里能看到注入的环境变量。

与 [[demo-device-plugin]] 那个通用 fake device 示例的关键不同：本 demo 学的是 **NVIDIA Device Plugin 的真实 SHAPE**——`Allocate` 只返回 envs，不直接 mount `/dev/nvidiaX`。在真实部署里，`/dev/nvidia0` 的注入交给 `nvidia-container-runtime` 的 prestart hook，按 env 中的 UUID 列表动态挂载（包括 driver 库版本对齐）。这种解耦是 NVIDIA 多年实战沉淀的设计，详见 [[gpu-scheduling-source]] 的「为什么 Allocate 不直接 mount /dev/nvidiaX」一节。

源码与部署清单见同目录其它文件，跑测步骤详见 [README](./README.md)。

## 设计要点

1. **GPU 风格命名**：资源名 `fake-gpu.k8s.io/gpu`、设备 ID `GPU-00000000-0000-0000-0000-00000000000{0..3}` 都模仿 NVIDIA 的形态，方便对照真实链路的日志和环境变量。

2. **Allocate 只返回 envs**：核心是 `NVIDIA_VISIBLE_DEVICES`——真实 NVIDIA 部署里这个 env 被 `nvidia-container-runtime` 的 prestart hook 读取，hook 据此把 `/dev/nvidiaN` 与 `libcuda.so` / `libnvidia-ml.so` 等驱动库 bind-mount 进容器。本 demo 集群里没有 NVIDIA runtime，但 env 依然会被 kubelet 合并进 CRI `CreateContainer`，容器内 `env` 命令就能看到——这本身就证明了「Allocate envs → CRI → 容器进程」这段链路通了，把它拼上「prestart hook → 真实设备挂载」就是生产 NVIDIA 部署。

3. **不带 NUMA / NVLink 拓扑**：本 demo 没填 `Device.Topology.Nodes`，`GetPreferredAllocationAvailable` 也为 false。真实 NVIDIA plugin 这两项都开了，DeviceManager 才能调 `GetPreferredAllocation` 让插件挑「同 NVLink 域的 4 张卡」给一个 4 卡 Pod。

4. **kubelet 重启重新注册**：与通用 fake device 示例一致，用 fsnotify 监听 `kubelet.sock` 重建事件。这是 production-ready 实现的基础要求，缺了会导致 kubelet 重启后 GPU capacity 突然变 0、所有 GPU Pod 调度失败。

## walkthrough：从 apply Pod 到容器看到 NVIDIA_VISIBLE_DEVICES

```
1) DaemonSet 起来, main.go 入口
   -> serve():
        rm /var/lib/kubelet/device-plugins/fake-gpu.sock
        net.Listen("unix", ".../fake-gpu.sock")
        grpc.NewServer(); RegisterDevicePluginServer; go server.Serve(lis)
        waitForSocket(...)
   -> register():
        grpc.Dial("unix:///var/lib/kubelet/device-plugins/kubelet.sock")
        RegistrationClient.Register({
            Version:"v1beta1",
            Endpoint:"fake-gpu.sock",
            ResourceName:"fake-gpu.k8s.io/gpu",
        })
   -> watcher.Add("/var/lib/kubelet/device-plugins")

2) kubelet 收到 Register, 反向 dial 我们的 sock
   kubelet -> 我们: GetDevicePluginOptions() -> {PreStartRequired:false}
   kubelet -> 我们: ListAndWatch()           -> stream
   我们   -> kubelet: ListAndWatchResponse{Devices: [
                GPU-00000000-0000-0000-0000-000000000000, Healthy,
                GPU-00000000-0000-0000-0000-000000000001, Healthy,
                GPU-00000000-0000-0000-0000-000000000002, Healthy,
                GPU-00000000-0000-0000-0000-000000000003, Healthy,
            ]}
   kubelet 更新内存 healthyDevices, 通过 statusManager 写 Node.status.capacity
        Node.status.capacity["fake-gpu.k8s.io/gpu"] = 4

3) kubectl apply pod-consumer.yaml
   resources.limits["fake-gpu.k8s.io/gpu"] = 1
   Scheduler -> NodeResourcesFit Filter -> 选中本 Node
   Scheduler -> Bind -> Pod.spec.nodeName = <node>

4) kubelet SyncLoop 收到 Pod ADD
   syncPod -> containerManager.Allocate -> DeviceManager.Allocate(pod, container)
        healthyDevices - allocatedDevices = 4 个空闲池
        挑出 ["GPU-00000000-0000-0000-0000-000000000000"]
        rpc 我们的插件: Allocate({DevicesIDs:["GPU-00000000-..."]})
   我们 -> kubelet: ContainerAllocateResponse{
        Envs: {
            NVIDIA_VISIBLE_DEVICES:     "GPU-00000000-...",
            NVIDIA_DRIVER_CAPABILITIES: "compute,utility",
            FAKE_GPU_DEVICES:           "GPU-00000000-...",
            FAKE_RESOURCE_NAME:         "fake-gpu.k8s.io/gpu",
        },
        Annotations: { "fake-gpu.k8s.io/allocated-uuids": "GPU-..." },
   }
   (注意: 没有 Mounts, 没有 Devices, 完全靠 env 注入. 这是 NVIDIA 风格.)

   kubelet podDevices.insert(pod, container, resource, ids, allocResp)
   kubelet 把 podDevices 序列化到 /var/lib/kubelet/device-plugins/kubelet_internal_checkpoint

5) kuberuntime.SyncPod
   PullImage busybox
   CreateContainer(Config{
        Envs: [..., NVIDIA_VISIBLE_DEVICES=GPU-..., ...],
   })
   -> 真实 NVIDIA 部署: nvidia-container-runtime 的 prestart hook 读 env,
                        bind-mount /dev/nvidia0 + driver libs 进容器 rootfs.
   -> 本 demo: 没有 hook, env 仍然进了容器, kubectl logs 可见.

6) 容器进程 execve 启动
   busybox sh -c 'env | grep NVIDIA_' -> 输出
        NVIDIA_VISIBLE_DEVICES=GPU-00000000-0000-0000-0000-000000000000
        NVIDIA_DRIVER_CAPABILITIES=compute,utility
   (验证 demo 跑通)
```

第 4 步是这条链路里最容易被忽略的关键：**Scheduler 只决定 Pod 调度到哪台 Node，「具体哪块 fake GPU 给这个容器」是 kubelet 侧 DeviceManager 与插件 Allocate 协商决定的**——这也是为什么 GPU 拓扑感知（NVLink、PCIe Switch、NUMA）必须在 kubelet 侧（或 DRA）解决，Scheduler 默认看不到设备粒度。

## 与生产级 NVIDIA Plugin 的差距

本 demo 故意简化的点，列出来作为面试 / 生产实现的参考：

- **NVML 枚举**：真实插件用 `github.com/NVIDIA/go-nvml/pkg/nvml` 调用 NVML API 拿 UUID、NUMA Node、温度、ECC 错误。本 demo 硬编码 4 个全零 UUID。
- **健康巡检**：真实 `ListAndWatch` 内部周期检查 XID、ECC、链路状态，不健康设备从上报列表移除并 `srv.Send` 新列表；本 demo 设备永远 Healthy。
- **拓扑信息**：真实插件 `Device` 消息会填 `TopologyInfo{Nodes: [{ID: numa}]}`，配合 Topology Manager 做 NUMA 对齐。本 demo 没填。
- **GetPreferredAllocation**：真实多卡场景下，把 8 张 GPU 中「同一 NVLink 域」的 4 张选给一个 4 卡 Pod 远优于随机选。本 demo 关闭了该能力。
- **Time-Slicing 配置**：真实插件可由 ConfigMap 把 1 张物理卡上报为 N 个虚拟卡，N 个 Pod 共享一卡（无显存隔离）。本 demo 不支持。
- **MIG 模式**：真实插件可在 MIG 已启用的 A100/H100 上把每个 MIG instance 上报为独立资源名（如 `nvidia.com/mig-3g.40gb`）。本 demo 不支持。
- **CDI 支持**：1.28+ 引入 CDI（Container Device Interface），`ContainerAllocateResponse.cdi_devices` 直接传 FQ 名让 runtime 按 CDI spec 处理。本 demo 仍用 envs 方式（与默认 NVIDIA plugin 一致）。
- **DRA 迁移**：未来 NVIDIA 会发布 DRA driver 替代这种 Device Plugin，本 demo 不涉及。
