#kubernetes #device-plugin #demo

相关笔记：[[kubelet-cri-source]] | [[gpu-scheduling]]

## 概述

本 demo 在 [[kubelet-cri-source]] 的「Device Plugin 机制」节落地为一个可运行项目。它实现一个虚拟的设备插件，向 kubelet 宣告 8 个名为 `learning-plan.io/fake-device` 的设备，让我们能在不需要真实 GPU 硬件的前提下走通整条链路：插件 Register、`ListAndWatch` 上报 → kubelet 把数量写入 Node `status.capacity` → Scheduler 把 Pod 调度上来 → kubelet 调插件 `Allocate` → 返回的 envs/mounts 合并进 CRI `CreateContainer` → 容器里能看到注入的环境变量和挂载点。

源码与部署清单见同目录其它文件，跑测步骤详见 [README](./README.md)。

## 设计要点

1. **两个 socket**：`/var/lib/kubelet/device-plugins/kubelet.sock` 是 kubelet 暴露的注册入口，由 kubelet 自己创建；`/var/lib/kubelet/device-plugins/learning-plan-fake-device.sock` 是我们插件自己创建并对外提供 `DevicePlugin` service 的 socket。DaemonSet 把这个目录用 hostPath 挂进容器，所以同一进程可以同时「连别人」和「被别人连」。

2. **Register 是单向通知**：`Register(version, endpoint, resourceName)` 调用成功后 kubelet 才会反向 dial 我们的 socket，依次调 `GetDevicePluginOptions` 和 `ListAndWatch`。如果在 Register 返回前自己的 server 还没起来，握手会失败。所以 `main.go` 是 `serve() -> register()` 顺序，并且 `serve` 内部 `waitForSocket` 等 listen 真正生效再返回。

3. **ListAndWatch 是长连接**：插件保持这个 stream 活着，状态有变化时 `srv.Send` 一次新的全量列表。本 demo 设备状态永不变化，所以仅用 60s 心跳「重发同一份」便于演示——真实插件不需要这个心跳，但需要在收到 `srv.Context().Done()` 时优雅退出。

4. **Allocate 入参是「已经选好的」deviceIDs**：DeviceManager 在 kubelet 侧从空闲池里挑出 N 个 ID（必要时先通过 `GetPreferredAllocation` 问插件偏好），再把这 N 个 ID 交给插件 `Allocate`。插件不参与「选哪个」，只参与「怎么把它注入容器」。

5. **kubelet 重启要重新注册**：kubelet 重启会重建 `kubelet.sock` 并清空内存中的插件 endpoint，旧的 Register 失效。`main.go` 用 fsnotify 监听 `pluginapi.DevicePluginPath` 目录（不是单个文件），看到 `kubelet.sock` Create 事件就 `stop -> serve -> register` 重来一次。

## walkthrough：从启动到容器拿到设备

```
1) Pod (DaemonSet) 启动, main.go 入口
   -> serve():
        rm /var/lib/kubelet/device-plugins/learning-plan-fake-device.sock
        net.Listen("unix", ".../learning-plan-fake-device.sock")
        grpc.NewServer(); RegisterDevicePluginServer; go server.Serve(lis)
        waitForSocket(...)
   -> register():
        grpc.Dial("unix:///var/lib/kubelet/device-plugins/kubelet.sock")
        RegistrationClient.Register({
            Version:"v1beta1",
            Endpoint:"learning-plan-fake-device.sock",
            ResourceName:"learning-plan.io/fake-device",
        })
   -> watcher.Add("/var/lib/kubelet/device-plugins")

2) kubelet 收到 Register, 在 ManagerImpl 内建立 endpoint, 反向 dial 我们的 sock
   kubelet -> 我们: GetDevicePluginOptions()  -> {PreStartRequired:false}
   kubelet -> 我们: ListAndWatch()            -> stream
   我们 -> kubelet:   ListAndWatchResponse{Devices: [fake-device-0..7, Healthy]}
   kubelet 更新内存 healthyDevices, 触发 statusManager 写 Node.status.capacity
        Node.status.capacity["learning-plan.io/fake-device"] = 8

3) 用户 kubectl apply 一个 Pod, resources.limits[learning-plan.io/fake-device]=2
   Scheduler -> NodeResourcesFit 插件 -> 选中本 Node
   API Server -> Pod.spec.nodeName = <node>

4) kubelet SyncLoop 收到 Pod ADD
   syncPod -> containerManager.Allocate -> ManagerImpl.Allocate(pod, container)
        从 healthyDevices 减去 allocatedDevices 得空闲池
        挑出 ["fake-device-0","fake-device-1"]
        rpc 我们的插件: Allocate({DevicesIDs:["fake-device-0","fake-device-1"]})
   我们 -> kubelet: ContainerAllocateResponse{
        Envs: { FAKE_VISIBLE_DEVICES: "fake-device-0,fake-device-1", ... },
        Mounts: [{ /etc/fake-devices <- /tmp/fake-devices, ReadOnly }],
        Devices: [],
        Annotations: { "learning-plan.io/allocated-ids": "fake-device-0,fake-device-1" },
   }
   kubelet 把这些字段合并进 CRI CreateContainer 的 ContainerConfig

5) kuberuntime.SyncPod
   RunPodSandbox -> CreateContainer (envs/mounts 注入) -> StartContainer
   containerd-shim-runc-v2 -> runc -> 容器进程 (env 中有 FAKE_VISIBLE_DEVICES)
```

第 4 步是这条链路里最容易被忽略的地方：**Scheduler 只决定 Pod 调度到哪个 Node，「具体哪两个 fake-device 给哪个容器」是 kubelet 侧 DeviceManager 与插件 Allocate 协商决定的**——这也是为什么 GPU 拓扑感知（NVLink、PCIe Switch）必须在 kubelet 侧解决，Scheduler 默认看不到设备粒度。

## 与生产级插件的差距

本 demo 故意简化了下面这些点，列出来作为面试与生产实现的参考：

- **健康巡检**：真实插件 `ListAndWatch` 内部会用厂商 SDK（NVML、XGMI、RDMA verbs）周期检查每个设备的 XID、ECC、温度、链路状态，把 Unhealthy 设备从上报列表移除并 `srv.Send` 新列表。本 demo 设备永远 Healthy。
- **拓扑信息**：`Device` 消息支持 `TopologyInfo` 字段填 NUMA Node ID，kubelet 的 Topology Manager 据此协调 CPU/Memory/Device 亲和。本 demo 没填。
- **GetPreferredAllocation**：真实多卡场景下，把 8 张 GPU 中「同一 NVLink 域」的 4 张选给一个 Pod 远优于随机选 4 张。生产插件会实现该 RPC。
- **CDI 支持**：1.28+ 引入 CDI（Container Device Interface）作为新的设备注入方式，`ContainerAllocateResponse.cdi_devices` 直接传 `vendor.com/gpu=gpudevice1` 这种 FQ 名，由运行时按 CDI spec 处理 mounts/devices。本 demo 仍用经典的 envs+mounts+devices 三元组。
- **重连退避**：`register()` 当前一次失败就退出，生产实现需要指数退避。
- **多版本协商**：`pluginapi.Version` 在 v1beta1 里是固定字符串 `"v1beta1"`，新版本（如 v1）发布后插件应支持版本协商。
