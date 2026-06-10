# 云原生 (Cloud Native) · 学习索引

云原生（Cloud Native）是 Kubernetes/容器编排岗位的核心考察领域，从 Docker 底层隔离技术到 K8s 控制面、网络、存储与扩展机制，是面试中考察广度与深度的主战场。

> ⬆ 返回 [知识库首页](../README.md)

## 🧭 推荐学习顺序

### 入门（建立容器与 K8s 全局认知）
Docker 概述与架构 → Docker 常用命令 → Dockerfile 最佳实践 → Cgroup 资源限制 → Namespace 资源隔离 → Union FS 联合文件系统 → Bridge 网络模式 → Kubernetes 基础架构 → Service 服务抽象 → Pod 探针 Probe

### 进阶（理解控制面与网络存储原理）
etcd 分布式存储 → OCI 容器运行时规范 → Kubernetes API Resource 设计 → RESTful API 路由设计 → RBAC 权限控制 → Informer List-Watch 机制 → K8s 网络模型 → CNI 接口规范 → kube-proxy → Flannel / Calico → Service / Headless Service → Volume 生命周期 → CSI 存储接口 → NFS-CSI / 云厂商 CSI → Kubebuilder 自定义控制器 → Operator 模式

### 深入（高频深挖与生产实战）
Scheduler 调度流程（Assume/Bind）→ Kube-Scheduler 深度解析 → GPU 调度与 Device Plugin → Cilium eBPF 深挖 → CNI 排障 → Multus 多网卡 → CSI sidecar → CSI 排障 → Ceph-CSI / Longhorn / OpenEBS → Velero 备份恢复 → Buildah 大镜像上传 → Google Borg → K8s 面试题汇总（查漏补缺）

## 📚 笔记清单

### Docker

| 笔记 | 简介 | 难度 |
|------|------|------|
| [Docker 概述与架构](docker/docker-basics.md) | 讲解 Docker 基于 Cgroup/Namespace/UnionFS 的容器虚拟化原理及 Client-Daemon-Registry 架构 | 入门 |
| [Dockerfile 最佳实践](docker/dockerfile.md) | 讲解精简镜像、利用构建缓存、多阶段构建与安全性等 Dockerfile 编写原则 | 入门 |
| [Docker 常用命令](docker/docker-commands.md) | 汇总镜像管理、容器生命周期等高频 Docker CLI 命令速查 | 入门 |
| [Buildah 大镜像上传方案](docker/buildah-large-image.md) | 实战讲解大模型镜像（20G）分片上传、断点续传与 buildah 加载推送的工程方案 | 深入 |
| [Cgroup 资源限制](docker/cgroup.md) | 讲解 Cgroup 通过层级树和子系统对进程的 CPU/内存/IO 进行资源控制与监控 | 进阶 |
| [Namespace 资源隔离](docker/namespace.md) | 讲解 Linux Namespace 为进程组提供独立资源视图，实现 PID/Net/Mnt 等容器隔离 | 进阶 |
| [Union FS 联合文件系统](docker/union-fs.md) | 讲解联合文件系统的分层只读/可写 branch 机制，是 Docker 镜像分层的底层原理 | 进阶 |
| [Docker Bridge 网络模式](docker/network-bridge.md) | 讲解默认 bridge 模式下 docker0 网桥、veth pair 与 iptables NAT 端口映射原理 | 入门 |
| [Docker Underlay 网络模式](docker/network-underlay.md) | 讲解 underlay 模式直接使用宿主机物理网络，为容器分配可路由 IP 与同网络平面通信 | 进阶 |
| [Docker Null 网络模式](docker/network-null.md) | 讲解 --net=none 空网络模式：容器有独立 Network Namespace 但不做任何网络配置 | 入门 |

### K8s-基础设施

| 笔记 | 简介 | 难度 |
|------|------|------|
| [Kubernetes 基础架构](kubernetes/infrastructure/kubernetes-basics.md) | 讲解 K8s 整体 Master/Node 架构、核心组件与基本概念，是云原生学习的总纲入口 | 入门 |
| [etcd 分布式存储](kubernetes/infrastructure/etcd.md) | 讲解基于 Raft 的分布式 KV 存储，用于服务发现、配置共享与一致性保障 | 进阶 |
| [Google Borg 集群管理](kubernetes/infrastructure/google-borg.md) | 讲解 K8s 前身 Borg 的 Borgmaster/Borglet/Cell 架构与大规模集群调度思想 | 深入 |
| [OCI 容器运行时规范](kubernetes/infrastructure/oci-runtime.md) | 讲解 OCI image-spec/runtime-spec 容器镜像与运行时标准及 containerd/runC 关系 | 进阶 |

### K8s-控制面

| 笔记 | 简介 | 难度 |
|------|------|------|
| [K8s API Resource 与 GVK](kubernetes/control-plane/api-resource.md) | 讲解 K8s API 资源的 Group/Version/Kind 层次结构与命名空间/集群作用域分类 | 进阶 |
| [Informer List-Watch 机制](kubernetes/control-plane/informer.md) | 讲解 client-go Informer 的 Reflector/DeltaFIFO/Indexer 协同与本地缓存机制 | 深入 |
| [K8s RESTful API 设计](kubernetes/control-plane/restful-api-design.md) | 讲解 K8s 核心组/非核心组、集群/命名空间资源的 RESTful API 路由格式 | 进阶 |
| [RBAC 权限控制](kubernetes/control-plane/rbac.md) | 讲解 Role/ClusterRole/RoleBinding/ClusterRoleBinding 四对象与主体绑定鉴权机制 | 进阶 |
| [Scheduler Assume 调度阶段](kubernetes/control-plane/scheduler-assume.md) | 讲解调度器 Filtering/Scoring/Assume/Bind 流程及 Assume 乐观绑定的设计意义 | 深入 |
| [Kube-Scheduler 深度解析](kubernetes/control-plane/scheduler-deep-dive.md) | 深度剖析调度器架构、Scheduling/Binding Cycle、调度框架插件与抢占机制 | 深入 |
| [K8s GPU 调度与 Device Plugin](kubernetes/control-plane/gpu-scheduling.md) | 讲解 Extended Resource、Device Plugin gRPC 接口与 GPU 不可压缩资源调度机制 | 深入 |
| [Pod 探针 Probe](kubernetes/control-plane/probes.md) | 讲解 liveness/readiness/startup 三种探针的失败后果、用途与对 Service 流量的影响 | 入门 |

### K8s-网络

| 笔记 | 简介 | 难度 |
|------|------|------|
| [K8s 网络模型](kubernetes/networking/network-model.md) | 讲解 K8s 网络三大原则（Pod-to-Pod/Service/External）与 IP-per-Pod 约束 | 进阶 |
| [CNI 接口规范](kubernetes/networking/cni.md) | 讲解 CNI 标准接口（ADD/DEL/CHECK）、工作流程与主流插件对比 | 进阶 |
| [kube-proxy 服务转发](kubernetes/networking/kube-proxy.md) | 讲解 kube-proxy watch Service/EndpointSlice 并维护 iptables/IPVS Service 数据面的职责边界 | 进阶 |
| [Calico CNI](kubernetes/networking/calico.md) | 讲解 Calico 的 Felix/BIRD 组件与 BGP/IPIP/VXLAN 三种跨节点通信及 NetworkPolicy | 进阶 |
| [Cilium eBPF CNI](kubernetes/networking/cilium.md) | 讲解 Cilium 基于 eBPF 绕过 iptables、BPF map O(1) 查找与 L7 策略的高性能网络 | 深入 |
| [Cilium 深度解析](kubernetes/networking/cilium-deep-dive.md) | 讲解 Cilium endpoint、identity、BPF map、kube-proxy replacement 与 L7 策略协作模型 | 深入 |
| [CNI 排障路径](kubernetes/networking/cni-troubleshooting.md) | 按 Pod IP、跨节点、ClusterIP、DNS、NetworkPolicy 五层定位 Kubernetes 网络问题 | 深入 |
| [Flannel CNI](kubernetes/networking/flannel.md) | 讲解 Flannel 的 VXLAN 与 host-gw 后端原理、封装开销与适用环境对比 | 入门 |
| [Weave Net CNI](kubernetes/networking/weave.md) | 讲解 Weave Net 的 fastdp/sleeve 双通道与内置 IPsec 加密通信 | 进阶 |
| [Multus 多网卡 CNI](kubernetes/networking/multus.md) | 讲解 Multus meta-plugin 委派机制实现 Pod 多网卡，用于电信/边缘场景 | 深入 |
| [Service 服务抽象](kubernetes/networking/service.md) | 讲解 Service 通过 Label Selector 聚合 Pod 及 ClusterIP/NodePort/LoadBalancer 类型 | 入门 |
| [Headless Service](kubernetes/networking/headless-service.md) | 讲解 clusterIP=None 的无头服务通过 DNS 直接解析到各 Pod IP 的机制 | 进阶 |

### K8s-存储

| 笔记 | 简介 | 难度 |
|------|------|------|
| [CSI 存储接口](kubernetes/storage/csi.md) | 讲解 CSI 标准化存储插件接口、out-of-tree 机制及 provisioner/attacher 等组件 | 进阶 |
| [Volume 生命周期](kubernetes/storage/volume-lifecycle.md) | 串联 PVC、PV、StorageClass、VolumeAttachment、Attach、Stage、Publish 与回收流程 | 进阶 |
| [CSI Sidecar 体系](kubernetes/storage/csi-sidecars.md) | 讲解 external-provisioner、attacher、resizer、snapshotter、registrar 等 sidecar 如何把 K8s 对象转成 CSI RPC | 深入 |
| [CSI 排障路径](kubernetes/storage/csi-troubleshooting.md) | 按 PVC Pending、Attach、Mount、Resize、Snapshot 分层定位存储插件问题 | 深入 |
| [Ceph-CSI 分布式存储](kubernetes/storage/ceph-csi.md) | 讲解 Ceph 的 MON/OSD/MDS 架构与 ceph-csi 提供 RBD/CephFS 存储驱动 | 深入 |
| [Longhorn 轻量分布式存储](kubernetes/storage/longhorn.md) | 讲解 Rancher Longhorn 的 Manager/Engine/Replica 架构与中小集群块存储方案 | 进阶 |
| [OpenEBS 容器化存储](kubernetes/storage/openebs.md) | 讲解 OpenEBS CAS 理念与 Mayastor/cStor/Jiva/Local PV 多引擎选型 | 深入 |
| [NFS-CSI 共享存储](kubernetes/storage/nfs-csi.md) | 讲解 NFS CSI Driver 支持 ReadWriteMany 多 Pod 共享读写的典型场景与配置 | 入门 |
| [云厂商 CSI 驱动](kubernetes/storage/cloud-provider-csi.md) | 讲解 AWS EBS/EFS 等云厂商 CSI 与云存储深度集成、零运维及厂商锁定权衡 | 进阶 |

### K8s-扩展

| 笔记 | 简介 | 难度 |
|------|------|------|
| [Kubebuilder 自定义控制器](kubernetes/extension/kubebuilder.md) | 讲解基于 client-go 的自定义 Controller 运行原理与 Reflector/WorkQueue/Reconcile 流程 | 进阶 |
| [Operator 模式](kubernetes/extension/operator-pattern.md) | 讲解 Operator = CRD + Custom Controller，用声明式 Reconcile 自动化管理有状态应用 | 进阶 |
| [Velero 备份恢复](kubernetes/extension/velero.md) | 讲解 Velero 通过 Backup/Restore Controller 实现集群资源备份、PV 快照与迁移 | 进阶 |

### K8s-面试

| 笔记 | 简介 | 难度 |
|------|------|------|
| [K8s 面试题汇总](kubernetes/interview/k8s-interview.md) | 汇总 K8s 架构、核心概念等高频面试问答，用于系统查漏补缺 | 进阶 |

---
共 **47** 篇 · 入门 10 / 进阶 23 / 深入 14
