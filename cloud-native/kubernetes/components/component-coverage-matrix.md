#kubernetes #components #matrix

相关笔记：[[component-note-standard]] | [[k8s-development-roadmap]] | [[kube-apiserver-component]] | [[etcd-component]] | [[kubelet-component]] | [[kube-proxy-component]]

# Kubernetes 组件覆盖矩阵

## 概述

这张表用于确认 `cloud-native/kubernetes/components/` 是否覆盖完整。标准分两层：

- **Tier 1 官方标准组件**：严格对齐 Kubernetes 官方 Components 文档。
- **Tier 2 研发扩展组件**：不属于最小官方 components 清单，但属于 Kubernetes 研发、平台工程、生产排障和面试必学内容。

## 状态定义

| 状态 | 含义 |
| --- | --- |
| outline-done | 已有统一章节、源码入口、基础链路、排查和面试要点，但还没有具体源码执行路径 |
| deep-done | 已补充具体源码执行路径、事故排查和 Event TTL 说明 |
| pending | 还没有组件文件或内容不足 |

## Tier 1: 官方标准组件

| 分类 | 组件 | 文件 | 状态 | 深挖主题 |
| --- | --- | --- | --- | --- |
| Control Plane | kube-apiserver | [[kube-apiserver-component]] | deep-done | 一个 create Pod 请求如何写入 etcd |
| Control Plane | etcd | [[etcd-component]] | deep-done | 一次 Kubernetes 写入如何落到 Raft/MVCC |
| Control Plane | kube-scheduler | [[kube-scheduler-component]] | deep-done | 一个 Pending Pod 如何经过 Filter/Score/Bind |
| Control Plane | kube-controller-manager | [[kube-controller-manager-component]] | deep-done | Deployment 如何创建/滚动 ReplicaSet |
| Control Plane | cloud-controller-manager | [[cloud-controller-manager-component]] | deep-done | LoadBalancer Service 如何创建云 LB |
| Node | kubelet | [[kubelet-component]] | deep-done | kubelet 如何拉起一个容器 |
| Node | kube-proxy | [[kube-proxy-component]] | deep-done | Service/EndpointSlice 如何变成节点转发规则 |
| Node | container runtime | [[container-runtime-component]] | deep-done | CRI RunPodSandbox/CreateContainer 如何落到 OCI runtime |
| Addon | DNS | [[coredns-component]] | deep-done | Service DNS 查询如何命中 CoreDNS cache |
| Addon | Web UI | [[kubernetes-dashboard-component]] | deep-done | Dashboard 请求如何带身份访问 apiserver |
| Addon | Container Resource Monitoring | [[metrics-server-component]] | deep-done | metrics-server 如何 scrape kubelet 并服务 HPA |
| Addon | Cluster-level Logging | [[logging-agent-component]] | deep-done | Fluent Bit 如何采集容器日志并补 metadata |

## Tier 2: 研发扩展组件

| 分类 | 组件 | 文件 | 状态 | 深挖主题 |
| --- | --- | --- | --- | --- |
| Networking | CNI plugin | [[cni-plugin-component]] | deep-done | runtime 调 CNI ADD 如何创建 Pod 网络 |
| Networking | Ingress Controller | [[ingress-controller-component]] | deep-done | Ingress 如何生成 NGINX/Envoy 配置并 reload |
| Storage | CSI driver | [[csi-driver-component]] | deep-done | PVC 如何经过 sidecar 到 NodePublishVolume |
| Admission | Admission Webhook | [[admission-webhook-component]] | deep-done | Mutating/Validating webhook 如何被 apiserver 调用 |
| Resource | Device Plugin | [[device-plugin-component]] | deep-done | GPU 设备如何注册、上报、Allocate 到容器 |

## 不纳入 components 的常见对象

| 名称 | 是否纳入 | 原因 |
| --- | --- | --- |
| kubectl | no | CLI 工具，不是集群运行组件 |
| kubeadm | no | 安装/初始化工具，不是长期运行组件 |
| Helm | no | 包管理/交付工具，不是 Kubernetes 组件 |
| Argo CD | no | GitOps 交付系统，属于平台生态 |
| Prometheus | no | 常见监控系统，不是官方 components 文档中的组件 |
| Grafana | no | 可视化工具，不是 Kubernetes 运行组件 |
| cert-manager | no | 常用 addon，但不是官方标准组件；可单独放扩展专题 |
| Gateway API controller | pending | 属于入口流量扩展，可在 Ingress/Gateway 专题中单独扩 |

## 后续扩展候选

如果继续扩展，不再打散“所有生态产品”，只按下面优先级加：

| 优先级 | 组件 | 原因 |
| --- | --- | --- |
| P1 | Gateway API Controller | Gateway API 正在替代部分 Ingress 场景 |
| P1 | cert-manager | webhook、controller、证书生命周期都很典型 |
| P2 | ExternalDNS | DNS controller 模式清晰，和 Ingress/Service 强相关 |
| P2 | Cluster Autoscaler / Karpenter | 调度、节点扩缩容、云 API 结合紧密 |
| P2 | Node Problem Detector | Node condition 和事件上报典型 addon |

## 面试要点

### Q: Kubernetes 官方标准组件有哪些？

A: Control Plane 包括 kube-apiserver、etcd、kube-scheduler、kube-controller-manager、cloud-controller-manager；Node 包括 kubelet、kube-proxy、container runtime；addon 包括 DNS、Web UI、资源监控和集群日志。

### Q: CNI/CSI/Device Plugin 为什么不放 Tier 1？

A: 它们是 Kubernetes 扩展接口或生态组件，不是官方 Components 文档里的最小运行组件分类。但生产集群必须依赖 CNI，存储和 GPU 场景也必须掌握 CSI/Device Plugin。

### Q: kubectl 算组件吗？

A: 不算本目录里的运行组件。kubectl 是客户端工具，重要但不属于 control plane、node component 或 addon。

### Q: 为什么 Dashboard 标成 legacy？

A: 原 Kubernetes Dashboard 仓库已在 2026-01-21 归档，缺少持续维护。它适合学习历史 Web UI 和 RBAC 风险，不建议作为新平台默认选型。

### Q: 后续新增组件按什么标准？

A: 必须满足三个条件之一：官方 components、生产常驻 addon、Kubernetes 研发扩展点。否则不放进 components 目录。
