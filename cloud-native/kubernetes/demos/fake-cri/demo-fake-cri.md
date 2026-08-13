#kubernetes #cri #kubelet #demo

相关笔记：[[cri-source]] | [[kubelet-cri-source]] | [[cni-source]] | [[oci-runtime]] | [[k8s-development-roadmap]]

## 概述

本 demo 在 [[cri-source]] 第九节落地为一个**最小可运行的 fake CRI runtime server**。它在 `/tmp/fake-cri.sock` 上监听 gRPC，只实现 CRI `RuntimeService` 的一个最小 RPC 子集，目标是用最少的代码讲清楚一件事：**kubelet 与容器运行时之间到底靠哪几个 RPC 对话，少了哪个 Node 就进不了 Ready**。

它**不能跑真业务容器**——没有实现 `CreateContainer` / `StartContainer`，也没有 ImageService。它的价值在于把 CRI 协议的「骨架」单独拎出来，让你能用 `crictl` 一条条命令去敲、去看 gRPC 的请求/响应，而不被 containerd 那一大堆 shim / snapshotter / content store 的细节淹没。运行与 `crictl` 探测步骤见 [README](./README.md)。

## 实现了哪些 RPC

| RPC | 行为 | 对应 CRI 概念 |
|-----|------|--------------|
| `Version` | 返回固定 `fake-cri v0.0.1` | kubelet 启动时握手，确认 runtime 在线 |
| `Status` | `RuntimeReady=true` / `NetworkReady=true` | **这两个 true 是 Node 进入 NodeReady 的前提** |
| `RunPodSandbox` | 内存 map 登记 sandbox，返回伪 id | 真实 runtime 这里要起 pause 容器 + 调 CNI |
| `StopPodSandbox` | 标记 NOTREADY（idempotent） | 删除流程的第一步 |
| `RemovePodSandbox` | 从 map 删除（idempotent） | GC 兜底，找不到也返回成功 |
| `PodSandboxStatus` | 单个 sandbox 详情，含假 IP `10.244.0.42` | `crictl inspectp` 看到的 IP 来自这里 |
| `ListPodSandbox` | 全量返回 | `crictl pods` 的数据源 |
| 其它 RPC | `UnimplementedRuntimeServiceServer` → `codes.Unimplemented` | CRI 是同步 unary，客户端立刻拿到错误码 |

## 设计要点

1. **「三个 RPC 让 kubelet 进 NodeReady」**：`Version`（握手）+ `Status`（`RuntimeReady`/`NetworkReady`）+ `ListPodSandbox`（同步已有 sandbox）。这条最小集合跑通，kubelet 就认为 runtime 健康。理解这点，排查「Node NotReady」时就知道先去看 `crictl info` / runtime 的 `Status` 返回。

2. **RPC 必须幂等**：`StopPodSandbox` / `RemovePodSandbox` 找不到目标也返回成功。kubelet 的 syncLoop 会重复触发删除（PLEG 事件重放、重启重放），非幂等会导致删除流程卡死。这正是 [[cri-source]] § 八强调的点。

3. **unix socket + gRPC unary**：server 用 `grpc.NewServer()` 监听 unix socket，客户端（kubelet / crictl）用 `grpc.DialContext("unix://...")` 连接。CRI 全部是同步 unary 调用，没有 streaming（除了 `Exec`/`Attach`/`PortForward` 走单独的 streaming server）。

4. **sandbox 只在内存**：`RunPodSandbox` 只往 map 里写状态，重启即丢。真实 containerd 用 bolt-db 持久化 + label 反查，kubelet 重启后还能把容器认回来。

## walkthrough：crictl 探测一个 sandbox 的生命周期

```
1) ./fake-cri  → listen unix:///tmp/fake-cri.sock

2) crictl version
   crictl --runtime-endpoint unix:///tmp/fake-cri.sock version
   → Version RPC → RuntimeName: fake-cri, RuntimeVersion: v0.0.1

3) crictl pods          (初始空)
   → ListPodSandbox RPC → 空列表

4) crictl runp pod.json (创建 sandbox)
   → RunPodSandbox RPC → 内存登记 → 返回 sandbox-<ts>-1
   crictl pods → 现在能看到 STATE=Ready 的 demo-pod
   crictl inspectp <id> → 伪 IP 10.244.0.42  (来自 PodSandboxStatus)

5) crictl stopp <id>    → StopPodSandbox → 标记 NOTREADY
   crictl rmp   <id>    → RemovePodSandbox → 从 map 删除
   (再删一次也成功 —— idempotent)
```

第 2 步 `Status` 返回的两个 `true` 是整条链路里最容易被忽略却最关键的：真实环境里 `NetworkReady=false` 往往意味着 CNI 没装好，Node 就一直 NotReady。

## 与真实运行时的差距

| 维度 | 这个 fake | containerd CRI plugin |
|------|-----------|------------------------|
| sandbox 持久化 | 内存 map，重启丢 | bolt-db + label 反查 |
| 真容器进程 | 无 | shim → runc create/start |
| 网络 | 假 IP 字符串 | CNI ADD/DEL（见 [[cni-source]]） |
| 镜像 | 无 ImageService | content store + snapshotter |
| Exec/Attach | 无 | streaming HTTP server |

要把它扩展成能跑真容器，需要补 `PullImage`（本地解压 tar）、`CreateContainer`（生成 OCI `spec.json`，见 [[oci-runtime]]）、`StartContainer`（fork+exec runc），那就成了一个最简化的「基于 runc 的 CRI 运行时」。

## 面试要点

### 高频问题

**Q: kubelet 和容器运行时之间是什么协议？最少需要哪几个 RPC 就能让 Node Ready？**
A: 通过 **CRI（Container Runtime Interface）**，基于 gRPC over unix socket，分 `RuntimeService` 和 `ImageService` 两组。让 kubelet 认为 runtime 健康的最小集合是 `Version`（握手）、`Status`（返回 `RuntimeReady=true` / `NetworkReady=true`）、以及 `ListPodSandbox`（同步已有状态）。

**Q: 为什么 `StopPodSandbox` / `RemovePodSandbox` 必须幂等？**
A: kubelet 的 syncLoop 会重复触发删除（PLEG 事件重放、kubelet 重启后重新对账）。如果非幂等，找不到 sandbox 就报错，会让整个 Pod 删除流程卡在 Terminating。所以约定：目标不存在也返回成功。

**Q: `Status` RPC 里 `NetworkReady=false` 通常意味着什么？**
A: 通常是 CNI 插件没装好或配置缺失（`/etc/cni/net.d` 为空）。kubelet 检测到 `NetworkReady=false` 会让 Node 保持 NotReady，新 Pod 无法调度上来。

**Q: CRI 调用是同步还是异步？没实现的 RPC 客户端会收到什么？**
A: CRI 绝大多数是同步 unary 调用（`Exec`/`Attach`/`PortForward` 例外，走独立 streaming server）。未实现的 RPC 通过 `UnimplementedRuntimeServiceServer` 返回 `codes.Unimplemented`，客户端立即拿到错误码。

**Q: 一个真实的 `RunPodSandbox` 除了登记状态还做了什么？**
A: 起一个 pause（infra）容器持有 Pod 的 network namespace，然后调用 CNI ADD 给这个 netns 配 IP/路由，之后同 Pod 的业务容器都 join 这个 netns，从而共享网络。fake 版省略了这两步，只返回一个假 IP。

### 面试加分点

- 能区分 `RuntimeService`（sandbox/container 生命周期）与 `ImageService`（拉取/管理镜像）两组接口
- 知道 sandbox = pause 容器 + netns，是「同 Pod 容器共享网络」的实现基础
- 理解 PLEG（Pod Lifecycle Event Generator）通过周期性 `ListPodSandbox` + relist 发现容器状态变化
- 了解 containerd 用 bolt-db 持久化 sandbox 元数据，从而支持 kubelet 重启后对账
- 清楚 CRI 之上还有 CDI（Container Device Interface）和 DRA 等设备注入的演进方向
