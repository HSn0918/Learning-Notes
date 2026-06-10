#kubernetes #cni #cilium #ebpf

相关笔记：[[cni]] | [[network-model]] | [[kube-proxy]] | [[calico]] | [[cilium-deep-dive]] | [[cni-troubleshooting]]

## Cilium 概述

Cilium 是新一代 CNI 插件，基于 eBPF（extended Berkeley Packet Filter）技术，在 Linux 内核中直接处理网络逻辑，避免了 iptables 的性能瓶颈。

## eBPF 架构原理

### 为什么 eBPF 比 iptables 快

传统方案（Calico、kube-proxy）依赖 iptables/netfilter：

- 每个 Service 对应一组 iptables 规则
- 数据包需要逐条匹配规则链（O(n) 复杂度）
- 规则数量随 Service/Endpoint 增长线性膨胀
- 规则更新需要全量重写（lock + 序列化）

eBPF 方案：

- 程序编译为字节码，注入内核的 hook 点（XDP、tc、socket）
- 使用 BPF map（hash table）存储路由和 Service 信息，查找 O(1)
- 数据包在内核最早的处理阶段就被拦截和转发
- 支持 JIT 编译为原生机器码，接近 native 性能

```mermaid
graph TB
    subgraph "传统 iptables 路径"
        A1[网卡收包] --> B1[Netfilter PREROUTING]
        B1 --> C1[逐条匹配 iptables 规则<br/>O&#40;n&#41; 复杂度]
        C1 --> D1[DNAT/SNAT 转换]
        D1 --> E1[路由决策]
        E1 --> F1[Netfilter POSTROUTING]
        F1 --> G1[发出数据包]
    end

    subgraph "Cilium eBPF 路径"
        A2[网卡收包] --> B2[XDP hook<br/>内核最早处理点]
        B2 --> C2[BPF map 查找<br/>O&#40;1&#41; 复杂度]
        C2 --> D2[直接转发/负载均衡]
        D2 --> G2[发出数据包]
    end

    style B2 fill:#2ecc71,color:#000
    style C2 fill:#2ecc71,color:#000
    style C1 fill:#e74c3c,color:#fff
```

## 核心组件

| 组件 | 部署方式 | 作用 |
| --- | --- | --- |
| **cilium-agent** | DaemonSet（每个节点） | 核心数据面组件，管理 eBPF 程序的加载/更新，监听 K8s API 同步 Pod、Service、NetworkPolicy 信息 |
| **cilium-operator** | Deployment（1-2 副本） | 集群级别的控制面，负责 IPAM（IP 分配管理）、垃圾回收、与云厂商 API 集成 |
| **Hubble** | DaemonSet + Deployment | 可观测性平台，提供网络流量可视化、Service Map、L7 协议监控 |
| **Hubble UI** | Deployment | Web 界面，展示 Service 之间的调用拓扑和流量详情 |
| **Cilium CLI** | 客户端工具 | 用于安装、状态检查、连通性测试 |

## 网络策略：L3/L4/L7

Cilium 的 NetworkPolicy 比 Kubernetes 原生 NetworkPolicy 更强大，支持到应用层（L7）的精细控制。

### L3/L4 策略

等同于 K8s 原生 NetworkPolicy：

```yaml
apiVersion: cilium.io/v2
kind: CiliumNetworkPolicy
metadata:
  name: allow-frontend-to-backend
spec:
  endpointSelector:
    matchLabels:
      app: backend
  ingress:
    - fromEndpoints:
        - matchLabels:
            app: frontend
      toPorts:
        - ports:
            - port: "8080"
              protocol: TCP
```

### L7 策略

Cilium 独有，可以控制 HTTP 方法和路径：

```yaml
apiVersion: cilium.io/v2
kind: CiliumNetworkPolicy
metadata:
  name: allow-get-only
spec:
  endpointSelector:
    matchLabels:
      app: api-server
  ingress:
    - fromEndpoints:
        - matchLabels:
            app: frontend
      toPorts:
        - ports:
            - port: "80"
              protocol: TCP
          rules:
            http:
              - method: GET
                path: "/api/v1/.*"
```

上面的策略表示：只允许 `frontend` Pod 以 GET 方法访问 `api-server` 的 `/api/v1/` 前缀路径，其他方法（POST、DELETE）和路径都会被拒绝。

## Service Mesh（无 Sidecar 方案）

传统 Service Mesh（如 Istio）在每个 Pod 中注入 Envoy sidecar，带来额外开销：

| 对比项 | Istio Sidecar 模式 | Cilium Service Mesh |
| --- | --- | --- |
| 代理位置 | 每个 Pod 内的 Envoy 容器 | 内核中的 eBPF 程序 |
| 额外延迟 | 每次请求多 2 次用户态/内核态切换 | 在内核中直接处理，几乎零额外延迟 |
| 资源开销 | 每个 Pod 额外 ~50MB 内存、~0.1 CPU | 共享节点级 eBPF 程序，开销极小 |
| L7 能力 | 完整（重试、限流、熔断、mTLS） | 支持基础 L7（负载均衡、路由、限流），复杂场景可集成 Envoy |
| 运维复杂度 | 高（sidecar 注入、版本管理） | 低（CNI 层面统一管理） |

Cilium 的思路：对于大多数场景，不需要完整的 sidecar，在内核层面用 eBPF 就能完成 L4 负载均衡和基础 L7 路由；对于需要复杂 L7 功能（如 mTLS、高级流量管理）的场景，Cilium 支持以 per-node 方式运行 Envoy（而非 per-pod），减少资源消耗。

## 安装与配置

```bash
# 安装 Cilium CLI
CILIUM_CLI_VERSION=$(curl -s https://raw.githubusercontent.com/cilium/cilium-cli/main/stable.txt)
curl -L --fail --remote-name-all \
  https://github.com/cilium/cilium-cli/releases/download/${CILIUM_CLI_VERSION}/cilium-linux-amd64.tar.gz
sudo tar xzvfC cilium-linux-amd64.tar.gz /usr/local/bin

# 安装 Cilium 到集群（使用 Helm）
helm repo add cilium https://helm.cilium.io/
helm install cilium cilium/cilium --version 1.15.0 \
  --namespace kube-system \
  --set kubeProxyReplacement=true \
  --set hubble.enabled=true \
  --set hubble.relay.enabled=true \
  --set hubble.ui.enabled=true

# 验证安装状态
cilium status --wait

# 运行连通性测试
cilium connectivity test
```

### 关键配置项

| 配置 | 说明 |
| --- | --- |
| `kubeProxyReplacement=true` | 完全替代 kube-proxy，使用 eBPF 实现 Service 负载均衡 |
| `hubble.enabled=true` | 启用 Hubble 可观测性 |
| `tunnel=vxlan` / `tunnel=disabled` | VXLAN 封装模式 / Native Routing 直连模式 |
| `ipam.mode=kubernetes` | 使用 K8s 原生 IPAM 分配 Pod IP |
| `bpf.masquerade=true` | 使用 eBPF 做 SNAT，替代 iptables MASQUERADE |

## 面试要点

**Q: Cilium 为什么比传统 CNI 性能更好？**

A: Cilium 使用 eBPF 直接在 Linux 内核中处理网络数据包，绕过了 iptables 和 netfilter 的逐条规则匹配。传统方案中 iptables 规则数量与 Service 数量成正比（O(n)），在大规模集群中会成为瓶颈；eBPF 使用 hash map 查找，时间复杂度为 O(1)，且避免了用户态/内核态的频繁切换。

**Q: Cilium 的 Hubble 是什么？有什么用？**

A: Hubble 是 Cilium 内置的可观测性平台，利用 eBPF 在内核层面采集网络流量元数据，提供：L3/L4/L7 的流量可视化、DNS 请求监控、HTTP/gRPC 调用追踪、NetworkPolicy 命中/丢弃日志。它不需要 sidecar，零侵入地提供 Service Map 和流量拓扑。

**Q: Cilium 如何实现 Service Mesh？和 Istio sidecar 模式有什么区别？**

A: Cilium 通过 eBPF 在内核层面实现 L7 流量管理（负载均衡、重试、限流），不需要在每个 Pod 注入 sidecar proxy。优势是：减少了每个 Pod 额外的 CPU/内存开销、消除了 sidecar 带来的额外网络跳数（latency）、简化了运维复杂度。
