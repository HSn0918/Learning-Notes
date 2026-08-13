#kubernetes #component #node #networking #service

相关笔记：[[k8s-development-roadmap]] | [[kube-proxy]] | [[service]] | [[network-model]] | [[cni]] | [[cilium-deep-dive]] | [[cni-troubleshooting]] | [[cni-plugin-component]] | [[k8s-interview]]

# kube-proxy

## 概述

`kube-proxy` 把 Kubernetes Service 抽象落到每个节点的数据面规则里。它 watch Service 和 EndpointSlice，然后维护 iptables/IPVS 规则，让访问 ClusterIP、NodePort、LoadBalancer VIP 的流量转发到后端 Pod。

核心边界：**CNI 负责 Pod 网络，kube-proxy 负责 Service 转发。**

## 职责边界

| 职责 | 说明 |
| --- | --- |
| service watch | 监听 Service 变化 |
| endpoint watch | 监听 EndpointSlice 变化 |
| datapath sync | 维护 iptables 或 IPVS 规则 |
| node local proxy | 每个节点独立维护 Service 转发表 |
| mode support | iptables、IPVS，或被 eBPF CNI 替代 |

## 核心链路

```mermaid
flowchart LR
    Client[client pod] --> VIP[service VIP]
    VIP --> Rule[kube-proxy rules]
    Rule --> EP[endpoint selection]
    EP --> Pod[backend pod]
```

## 关键机制

- ClusterIP 是虚拟 IP，不属于某个真实网卡。
- kube-proxy 不做七层路由，只做四层转发。
- iptables 模式靠 NAT 链转发，规则数量随 Service/Endpoint 增长。
- IPVS 模式用 Linux IPVS virtual server 和 real server。
- Cilium 等 eBPF 数据面可以替代 kube-proxy，但 Service API 仍然存在。

## 源码导读

| 目标 | 源码位置 | 读什么 |
| --- | --- | --- |
| 进程入口 | `cmd/kube-proxy/app/server.go` | `NewProxyCommand` |
| Linux proxier 选择 | `cmd/kube-proxy/app/server_linux.go` | iptables/IPVS/nftables proxier 初始化 |
| iptables proxier | `pkg/proxy/iptables/proxier.go` | `NewProxier`、`syncProxyRules` |
| IPVS proxier | `pkg/proxy/ipvs/proxier.go` | virtual server / real server 同步 |
| nftables proxier | `pkg/proxy/nftables/proxier.go` | nftables ruleset |
| 公共 service map | `pkg/proxy/servicechangetracker.go` | Service change tracking |
| EndpointSlice map | `pkg/proxy/endpointslicecache.go` | EndpointSlice 聚合 |

同步链路：

```text
kube-proxy startup
  -> create proxier by mode
  -> watch Service
  -> watch EndpointSlice
  -> change tracker updates local maps
  -> bounded frequency runner calls syncProxyRules
  -> write iptables / IPVS / nftables rules
```

精简源码骨架：

```go
func (proxier *Proxier) OnServiceUpdate(oldSvc, newSvc *v1.Service) {
    proxier.serviceChanges.Update(oldSvc, newSvc)
    proxier.Sync()
}

func (proxier *Proxier) syncProxyRules() error {
    services := proxier.serviceMap
    endpoints := proxier.endpointsMap
    rules := buildRules(services, endpoints)
    return proxier.dataplane.Apply(rules)
}
```

## 深入：Service/EndpointSlice 如何变成节点转发规则

这条链路回答一个具体问题：**创建一个 ClusterIP Service 后，每个节点上的 kube-proxy 如何把 Service VIP 转成 iptables/IPVS/nftables 规则？**

### 0. 前置条件

| 前置项 | 说明 |
| --- | --- |
| Service 已存在 | apiserver 中有 Service spec，包括 ClusterIP、ports、type |
| EndpointSlice 已生成 | EndpointSlice controller 根据 selector 维护后端 Pod 地址 |
| kube-proxy 已同步 cache | Service 和 EndpointSlice informer 都完成初始同步 |
| 数据面模式确定 | iptables、IPVS、nftables，或被 eBPF replacement 替代 |

核心边界：kube-proxy 只负责 Service 数据面规则；Pod IP、veth、跨节点路由由 CNI 负责。

### 1. Watch Service 和 EndpointSlice

源码入口：

- `pkg/proxy/iptables/proxier.go`
- `pkg/proxy/ipvs/proxier.go`
- `pkg/proxy/nftables/proxier.go`
- `pkg/proxy/servicechangetracker.go`
- `pkg/proxy/endpointschangetracker.go`

以 iptables proxier 为例：

```text
OnServiceAdd / OnServiceUpdate / OnServiceDelete
  -> serviceChanges.Update
  -> Sync

OnEndpointSliceAdd / OnEndpointSliceUpdate / OnEndpointSliceDelete
  -> endpointsChanges.EndpointSliceUpdate
  -> Sync
```

精简骨架：

```go
func (proxier *Proxier) OnServiceUpdate(oldService, service *v1.Service) {
    if proxier.serviceChanges.Update(oldService, service) && proxier.isInitialized() {
        proxier.Sync()
    }
}

func (proxier *Proxier) OnEndpointSliceUpdate(_, endpointSlice *discovery.EndpointSlice) {
    if proxier.endpointsChanges.EndpointSliceUpdate(endpointSlice, false) && proxier.isInitialized() {
        proxier.Sync()
    }
}
```

### 2. ChangeTracker 合并变化

Service 和 EndpointSlice 事件可能很频繁，kube-proxy 不会每个事件都立即完整写一遍内核规则。ChangeTracker 先记录变化，再在 `syncProxyRules` 中更新本地 map：

| 结构 | 作用 |
| --- | --- |
| `ServiceChangeTracker` | 记录 ServicePortMap 的增删改 |
| `EndpointsChangeTracker` | 记录 EndpointSlice 到 EndpointsMap 的变化 |
| `ServicePortMap` | 当前节点理解的所有 Service port |
| `EndpointsMap` | 当前节点理解的所有后端 endpoint |
| `BoundedFrequencyRunner` | 限制 sync 频率，合并抖动事件 |

### 3. `syncProxyRules` 构建数据面规则

源码入口：`pkg/proxy/iptables/proxier.go`

iptables 模式下，`syncProxyRules` 会：

```text
syncProxyRules
  -> apply serviceChanges to svcPortMap
  -> apply endpointChanges to endpointsMap
  -> ensure top-level chains
  -> build KUBE-SERVICES / KUBE-NODEPORTS / endpoint chains
  -> handle ClusterIP / ExternalIP / LoadBalancer / NodePort
  -> handle internalTrafficPolicy / externalTrafficPolicy
  -> write rules through iptables-restore
```

精简骨架：

```go
func (proxier *Proxier) syncProxyRules() error {
    proxier.svcPortMap.Update(proxier.serviceChanges)
    proxier.endpointsMap.Update(proxier.endpointsChanges)

    for svcName, svc := range proxier.svcPortMap {
        endpoints := proxier.endpointsMap[svcName]
        if len(endpoints) == 0 {
            writeRejectOrDropRules(svc)
            continue
        }
        writeServiceChains(svc)
        writeEndpointChains(endpoints)
        writeClusterIPRules(svc)
        writeNodePortRules(svc)
        writeLoadBalancerRules(svc)
    }

    return proxier.iptables.RestoreAll(proxier.filterRules, proxier.natRules)
}
```

IPVS 模式下思路类似，但输出从 iptables NAT 链变成 IPVS virtual server / real server；nftables 模式输出 nftables ruleset。

### 4. 数据包如何命中规则

以 ClusterIP 为例：

```text
client pod
  -> dst=<clusterIP>:<port>
  -> node netfilter PREROUTING/OUTPUT
  -> KUBE-SERVICES
  -> service chain
  -> endpoint chain
  -> DNAT to <podIP>:<targetPort>
  -> CNI datapath routes packet to backend Pod
```

这解释了一个常见误区：kube-proxy 不是每个包的用户态代理。它把规则写进内核，真正转发发生在内核和 CNI 数据面。

### 5. 失败点与排查映射

| 现象 | 对应阶段 | 先看哪里 |
| --- | --- | --- |
| Service 没 endpoints | EndpointSlice controller 或 selector | `kubectl get endpointslice`、Pod labels/readiness |
| 规则未生成 | kube-proxy watch/sync | kube-proxy logs、Service/EndpointSlice cache |
| ClusterIP 不通但 Pod IP 通 | kube-proxy 规则 | iptables/IPVS/nftables、conntrack |
| NodePort 不通 | NodePort 规则或外部网络 | host firewall、security group、`externalTrafficPolicy` |
| 只有本节点不通 | 节点本地规则或 CNI | 对比其他节点规则 |
| eBPF 模式看不到规则 | kube-proxy replacement | 查 Cilium/其他 eBPF map |

## 源码阅读重点

### kube-proxy 不代理每个包

源码里的 proxier 主要做规则同步。数据包命中 iptables/IPVS/nftables 后由内核转发，所以 CPU 热点不一定在 kube-proxy 进程上。

### ChangeTracker

Service 和 EndpointSlice 变化很频繁，kube-proxy 不应该每个事件都全量重建所有规则。ChangeTracker 记录增量，`syncProxyRules` 再按节流策略批量落盘。

### eBPF replacement

如果集群使用 Cilium kube-proxy replacement，`pkg/proxy/*` 这套代码可能完全不运行。排障时要先确认数据面模式，再决定看 iptables、IPVS 还是 BPF map。

## 故障信号

| 现象 | 常见方向 |
| --- | --- |
| ClusterIP 不通 | kube-proxy、EndpointSlice、iptables/IPVS、CNI |
| NodePort 不通 | nodePort、防火墙、externalTrafficPolicy |
| 只有 DNS 名不通 | CoreDNS，不一定是 kube-proxy |
| eBPF 模式查不到 iptables | kube-proxy replacement 正常现象 |

## 事故排查

### 先判断故障层级

Service 不通先分三段：

| 检查 | 结论 |
| --- | --- |
| Pod IP 直连不通 | 先转 CNI/Pod/NetworkPolicy |
| Pod IP 通、ClusterIP 不通 | 重点查 kube-proxy 数据面规则 |
| ClusterIP 通、DNS 名不通 | 转 CoreDNS |
| 只有 NodePort/LoadBalancer 不通 | 查外部网络、NodePort、防火墙、cloud LB |

### Event 保留时间

kube-proxy 本身很少生成对用户有用的 Pod event。Service/EndpointSlice 相关事件如果出现，也默认只保留 `1h`，由 kube-apiserver `--event-ttl` 控制。Service 事故更依赖对象快照、kube-proxy 日志、节点规则和抓包。

### 证据保全

| 证据 | 用途 |
| --- | --- |
| Service YAML | type、ports、trafficPolicy、sessionAffinity |
| EndpointSlice YAML | endpoint IP、ready/serving/terminating、zone/node |
| kube-proxy logs | sync 失败、iptables/ipvs 调用失败 |
| 节点规则快照 | `iptables-save`、`ipvsadm`、`nft list ruleset` |
| conntrack | 判断旧连接是否被 conntrack 影响 |
| 抓包 | 区分规则没命中、DNAT 后不通、回包路径异常 |

### 常见事故路径

1. `curl <clusterIP>` 不通时，先确认 EndpointSlice 里有 ready endpoint。没有 endpoint 时 kube-proxy 正常也会拒绝或丢弃。
2. `externalTrafficPolicy: Local` 下，没本地 endpoint 的节点可能无法接收外部流量，这是配置语义，不是 kube-proxy 坏了。
3. 大集群规则同步慢时，看 kube-proxy sync latency、iptables lock、EndpointSlice 数量和单 Service endpoint 数量。
4. 使用 Cilium kube-proxy replacement 时，不要按 iptables 模式排查，应查 eBPF service map。

## 排查命令

```bash
kubectl get svc,endpointslice -A
kubectl describe svc <service> -n <namespace>
kubectl get endpointslice -n <namespace> -l kubernetes.io/service-name=<service> -o yaml
kubectl -n kube-system logs ds/kube-proxy --tail=300
iptables-save -t nat | grep KUBE-SERVICES
iptables-save -t nat | grep <service>
ipvsadm -Ln
nft list ruleset
conntrack -L | grep <cluster-ip>
```

## 面试要点

### Q: kube-proxy 和 CNI 的区别？

A: CNI 解决 Pod 如何获得 IP、接入网络、跨节点互通；kube-proxy 解决 Service VIP 如何转发到后端 Pod。

### Q: ClusterIP 是真实 IP 吗？

A: 不是绑定在某张网卡上的真实 IP，而是由节点数据面规则识别并转发的虚拟服务地址。

### Q: kube-proxy 是否在流量路径上转发每个包？

A: iptables/IPVS 模式下 kube-proxy 主要负责下发规则，真正转发发生在内核数据面，不是每个包都经过 kube-proxy 用户态进程。

### Q: `externalTrafficPolicy: Local` 的作用？

A: 保留客户端源 IP，并避免跨节点二次转发；代价是只有存在本地 endpoint 的节点能接收流量。

### Q: eBPF 替代 kube-proxy 后 Service 还存在吗？

A: 存在。消失的是 kube-proxy 这个实现组件，Service API 和 EndpointSlice 仍然是控制面输入。
