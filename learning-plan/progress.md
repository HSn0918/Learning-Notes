#kubernetes #学习计划 #打卡

相关笔记：[[k8s-development-roadmap]] | [[hami-learning-path]] | [[learn-k8s-via-hami]]

## 用法

- 每完成一项把 `[ ]` 改成 `[x]`
- 每周末在「周末复盘」处写一两句话：画了什么图、卡在哪、读了多少行源码
- 卡住的题目记在最底部「待解决问题」，每周回顾一次

## 关于「6 周」——先读这段，别被节奏绑架

下面的「Week 1 ~ Week 6」是**逻辑顺序**，不是**日历周**。「6 周」对应的是**全职脱产、每天 4-6 小时**的下限投入。按你的实际情况换算：

| 你的情况 | 实际预期 | 每个「Week」实际花 |
| :--- | :--- | :--- |
| 全职脱产 4-6h/天 | 6-8 周 | 约 1-1.5 周 |
| 在职 1.5-2h/天 + 周末 | 10-14 周 | 约 2 周 |
| 在职碎片时间 | 4-6 个月 | 视情况 |

**两个偏紧的点要特别注意**：
- **Week 4（CSI）**：CSI 三 service + 6 sidecar + 端到端时序，新手吃透要 1.5-2 周，不是 1 周。
- **Week 5（etcd 源码）**：etcd 的 raft 实现单独给 2 周都不算多。「1 周读完 etcd 源码」是不现实的——能读懂 raft Ready 循环 + WAL/snapshot 关系就算达标，MVCC 细节可以二刷。

某周没按时完成 = 这张表对你的投入估得太乐观，**不是你太慢**。往后顺延，别因此否定自己，更别为了打勾而囫囵吞枣。

起点日期：____________  目标完成日期（按上表换算后填）：____________

---

## Week 1：Scheduler-framework + 自定义插件

> 目标：默写 11 扩展点顺序；解释 percentageOfNodesToScore / CycleState / Snapshot 设计动机。
> 预计：全职 ~1 周 / 在职 ~2 周

### 阅读

- [ ] [[scheduler-framework-source]] 第 1-3 节（整体架构 + 调度循环）
- [ ] [[scheduler-framework-source]] 第 4-6 节（CycleState / Snapshot / 队列）
- [ ] [[scheduler-framework-source]] 第 7-9 节（11 扩展点 + 自定义插件 + extender 对比）
- [ ] [[scheduler-deep-dive]] [[scheduler-assume]] 横向补
- [ ] 本地源码：`~/github/kubernetes/pkg/scheduler/scheduler.go` 主循环
- [ ] 本地源码：`~/github/kubernetes/pkg/scheduler/framework/interface.go` 11 扩展点定义

### 动手

- [ ] 跑通 [[demo-scheduler-plugin]]，能 `kubectl get pod -o wide` 看到 Pod 落在预期 Node
- [ ] **改造**：把 Score 插件改成"偏好带 `gpu-tier=high` label 的 Node"，部署验证
- [ ] 用 `kubectl logs` 看 secondary scheduler 的调度日志

### 周末复盘（默写）

- [ ] 默写 11 扩展点顺序：PreFilter → Filter → PostFilter → PreScore → Score → NormalizeScore → Reserve → Permit → PreBind → Bind → PostBind
- [ ] 画调度队列三态转换：activeQ ↔ backoffQ ↔ unschedulableQ
- [ ] 一句话解释：为什么 percentageOfNodesToScore 默认 50（节点多时降到更低）

**本周复盘笔记**：
```
（在这里写一两句话：画了什么图、卡在哪、读了多少行源码）
```

---

## Week 2：Device Plugin + GPU/HAMi

> 目标：解释 Scheduler 与 kubelet 在 GPU 分配中的分工；讲清 HAMi 4 块组件协作链路。
> 预计：全职 ~1 周 / 在职 ~2 周

### 阅读

- [ ] [[kubelet-cri-source]] 第 3 节（Device Plugin）
- [ ] [[gpu-scheduling-source]] 全篇
- [ ] [[gpu-scheduling]]（概念层）
- [ ] [[hami-source]] 全篇（HAMi 怎么实现 GPU 虚拟调度：webhook / extender / device-plugin / libvgpu）
- [ ] [[hami-learning-path]] 阶段 3-5（webhook / extender / device-plugin 源码）
- [ ] HAMi 真实仓库：`pkg/scheduler/webhook.go` + `pkg/scheduler/routes/` + `pkg/device-plugin/`

### 动手

- [ ] 跑 [[demo-fake-gpu]]（基础 SHAPE）
- [ ] 跑 [[demo-hami-mac]]（HAMi 风格 env 注入）
- [ ] 验证 `kubectl logs hami-consumer` 看到 `LD_PRELOAD` + `CUDA_DEVICE_MEMORY_LIMIT_0`
- [ ] 改造 [[demo-hami-mac]]：切片数从 10 改 4，看 Node capacity 变成 16
- [ ] 跑 `libvgpu-hook-demo/run-in-docker.sh` 看 malloc hook 拦截效果

### 周末复盘

- [ ] 不看原图，徒手画 [[learn-k8s-via-hami]] 开头的端到端图
- [ ] 答出：为什么 HAMi 选 extender 不选 framework plugin
- [ ] 答出：Scheduler 和 kubelet 在 GPU 分配中分别负责什么（提示：调到哪台 vs 哪块卡）

**本周复盘笔记**：
```
```

---

## Week 3：CRI 链路 + OCI Runtime

> 目标：画完整链路 `kubelet → CRI → containerd → containerd-shim → runc`；能用 crictl/ctr 单步排查。
> 预计：全职 ~1 周 / 在职 ~2 周

### 阅读

- [ ] [[kubelet-cri-source]] 第 1-2 节（SyncLoop / PLEG）
- [ ] [[kubelet-cri-source]] 第 4-5 节（CRI gRPC + containerd）
- [ ] [[cri-source]] 全篇（CRI proto 契约 / cri-client / sandbox+container / dockershim 移除）
- [ ] [[oci-runtime]]
- [ ] 本地源码：`~/github/kubernetes/pkg/kubelet/kubelet.go` 的 syncLoop
- [ ] 本地源码：`~/github/kubernetes/pkg/kubelet/pleg/` 

### 动手

- [ ] 跑 [[demo-fake-cri]]，用 `crictl` 探测最小 fake CRI server
- [ ] 在 kind 节点容器里跑 `crictl ps` / `crictl inspect <id>`
- [ ] 用 `crictl exec` 进容器验证 env
- [ ] 看一个真实容器的 OCI runtime spec：`docker exec kind-node crictl inspect <id> | jq '.info.runtimeSpec'`
- [ ] 用 `nsenter` 进 containerd-shim 进程看它和 runc 怎么交互
- [ ] 故意让一个容器 OOM，看 PLEG 怎么感知到、kubelet 怎么 restart

### 周末复盘

- [ ] 画 Pod 启动完整链路（包含 pause / Sandbox 容器作用）
- [ ] 答出：PLEG 为什么会 not healthy（提示：relist 周期、stop-the-world）
- [ ] 答出：1.27+ Evented PLEG 解决了什么

**本周复盘笔记**：
```
```

---

## Week 4：CSI 实战

> 目标：画 PVC → Provision → Attach → Mount 完整时序；改一个最小 CSI driver。
> 预计：全职 ~1.5-2 周 / 在职 ~3 周 ⚠️ 偏紧——CSI 三 service + 6 sidecar 内容量大，别赶。

### 阅读

- [ ] [[csi-source]] 全篇
- [ ] [[csi]]（概念）
- [ ] [[openebs]] [[longhorn]] [[ceph-csi]] [[nfs-csi]] 横向对比
- [ ] [[cloud-provider-csi]]

### 动手

- [ ] 跑 [[demo-csi-hostpath]]，apply PVC → 看绑定过程
- [ ] 用 `kubectl describe pvc` + `kubectl get volumeattachment` 看 4 个 sidecar 各自的动作
- [ ] **改造**：给 [[demo-csi-hostpath]] 加 VolumeExpand 能力
- [ ] 改一次 `volumeBindingMode: WaitForFirstConsumer`，对比看 PV 何时创建

### 周末复盘

- [ ] 画 4 个 sidecar 的职责（provisioner / attacher / resizer / snapshotter）
- [ ] 答出：CSI 三大 service（Identity / Controller / Node）分别在哪种 Pod 里跑
- [ ] 答出：WaitForFirstConsumer 改变了什么（提示：延后 PV 创建到 Pod 调度后）

**本周复盘笔记**：
```
```

---

## Week 5：etcd 源码 + 综合实战

> 目标：解释 raft Ready 循环、WAL+snapshot+compact、MVCC revision 模型；拼一个综合小项目。
> 预计：全职 ~2 周 / 在职 ~3-4 周 ⚠️ 偏紧——「读懂 etcd raft」单独 2 周不算多。达标线是 raft Ready 循环 + WAL/snapshot，MVCC 细节可二刷。

### 阅读

- [ ] [[etcd-source]] 全篇
- [ ] [[etcd]]（概念）
- [ ] 本地源码：`~/github/etcd/raft/` 的 raft 主循环
- [ ] 本地源码：`~/github/etcd/server/mvcc/` 的 KV revision 模型

### 动手

- [ ] 跑 [[demo-raftexample-walkthrough]]
- [ ] 用 `etcdctl` 实操：watch / lease / txn / compact / defrag
- [ ] 用 `etcdctl endpoint status` 看 leader、raft index、DB size
- [ ] **综合**：拼一个 GPU Operator：CRD + Reconciler + Webhook + 自定义 Scheduler scorer，对接 [[demo-hami-mac]] 的 plugin

### 周末复盘

- [ ] 画一次写请求完整路径（client → apiserver → etcd WAL → raft 同步 → apply → 回 client）
- [ ] 答出：MVCC 的 revision 是逻辑时钟还是物理时钟（提示：cluster-wide 单调递增）
- [ ] 答出：compact 和 defrag 的差异（一个清 revision、一个回收磁盘）

**本周复盘笔记**：
```
```

---

## Week 6：CNI + 面试冲刺

> 目标：理解 CNI ADD/DEL；说清 Calico vs Cilium 数据面差异；面试题过 2 遍。
> 预计：全职 ~1 周 / 在职 ~2 周

### 阅读

- [ ] [[cni-source]] § 1-4（CNI 协议 / libcni / bridge·host-local / containerd 调用时序）
- [ ] [[cni-source]] § 6（Calico / Cilium / Flannel 数据面入口对照）
- [ ] [[cni]] [[calico]] [[cilium]] [[flannel]] [[weave]] [[multus]]
- [ ] [[service]] [[network-model]] [[headless-service]]
- [ ] [[k8s-interview]] 全部题

### 动手

- [ ] 跑 [[demo-cni-bridge]] 的 `./run-in-docker.sh`，看两个 netns 通过 cni0 互 ping
- [ ] 读 `demos/cni-bridge/learning-bridge` 100 行 bash，对照 [[cni-source]] § 3 的真实 bridge 源码
- [ ] **改造**：给 [[demo-cni-bridge]] 加一个 portmap 链插件，模拟 hostport
- [ ] 在 kind 节点里看 iptables / ipvs 规则，对照 Service 配置

### 周末复盘 + 模拟面试

- [ ] 模拟 30 分钟技术分享：讲清 HAMi 端到端链路
- [ ] 再过一遍各源码导读笔记的「面试要点」章节
- [ ] 自检题目（下面那批）能在 5 分钟内答完

**本周复盘笔记**：
```
```

---

## 高阶自检题（6 周后应该全部能答）

1. 11 个 Scheduler 扩展点的顺序、各自能修改什么
2. 调度队列 activeQ / backoffQ / unschedulableQ 的转换条件
3. Framework plugin vs Extender 各自的适用场景
4. Scheduler assume 是干什么的、Bind 失败怎么回滚
5. Device Plugin 的 4 个 RPC，Allocate 返回值的 4 个字段
6. kubelet 重启后 Device Plugin 怎么重新连接（fsnotify 监听什么）
7. HAMi 为什么不能用 framework plugin 替代 extender
8. HAMi 在 1 张物理卡上跑 2 个 Pod，配额隔离的"信任边界"在哪
9. LD_PRELOAD 的 Linux 实现机制（ld.so 顺序、RTLD_NEXT）
10. PLEG 的工作原理、PLEG is not healthy 怎么排查
11. Evented PLEG 解决了什么
12. CRI 的 RunPodSandbox 和 CreateContainer 谁先调用、谁创建 pause 容器
13. 4 个 CSI sidecar 各自连哪个 CSI RPC
14. `volumeBindingMode: WaitForFirstConsumer` 改变了什么
15. CSI Controller / Node service 分别部署在哪种 Pod 里
16. etcd raft 的 Ready 循环、WAL 和 snapshot 的关系
17. compact 和 defrag 的差异
18. 一次写请求从 apiserver 到 etcd 落盘的完整路径
19. CNI 的 ADD / DEL 各自传哪些参数
20. Calico BGP / IPIP 和 Cilium eBPF 数据面的本质差异

---

## 待解决问题

> 看不懂的、卡住的、半懂半不懂的，先记在这里。每周末统一回顾。

- [ ] 
- [ ] 
- [ ] 

---

## 完成纪念

6 周收官时在这里贴：
- 最终的端到端 HAMi 链路图（自己画的）
- GPU Operator 综合项目的截图 / 仓库链接
- 一段不超过 200 字的总结：哪个机制最让你"原来如此"
