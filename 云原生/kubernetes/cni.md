------

# ✅ CNI + Calico + Pod通信排查 超实用面试速记（含命令）

------

## 🧩 一、CNI 是什么？

> CNI（Container Network Interface）是一种标准接口，用于为容器（Pod）**分配 IP、配置路由、连接网络**。

### 🛠 常见的 CNI 插件：

- Flannel：简单，VXLAN 封装
- **Calico**：功能最全，支持路由 + 策略控制
- Cilium：基于 eBPF，性能高
- Kube-OVN：支持 L2/L3，企业级方案

------

## 🐱 二、Calico 是什么？它的作用？

> **Calico 是一个强大的 CNI 插件**，不仅给 Pod 分配 IP，还支持 **跨节点通信、网络策略、安全隔离**。

### ✅ Calico 能做什么？

1. 给每个 Pod 分配可路由 IP（无 NAT）
2. 建立路由/封装隧道（BGP、IPIP、VXLAN）
3. 实现 NetworkPolicy（控制谁能访问谁）

------

### 🧱 Calico 的核心组件和作用：

| 组件          | 作用说明                                       |
| ------------- | ---------------------------------------------- |
| calico-cni    | 在 Pod 启动时设置网络连接（veth pair + IP）    |
| calico-node   | 每个节点的核心组件，包含 Felix、BGP 或隧道逻辑 |
| Felix         | 管理 iptables、防火墙规则，执行策略控制        |
| BIRD / GoBGP  | 节点间的路由同步（用于 BGP 模式）              |
| Typha（可选） | 减轻 etcd 压力，适用于大集群                   |

------

### 🌍 三种跨节点通信模式：

| 模式        | 原理                                      |
| ----------- | ----------------------------------------- |
| BGP（默认） | 各节点之间通过 BGP 协议同步路由，性能最佳 |
| IPIP        | IP-in-IP 封装，穿越网络不通的场景         |
| VXLAN       | 使用 VXLAN 封装数据帧，适用于云厂商环境   |

------

## 🌐 三、Pod 跨节点通信的完整流程（以 Calico 为例）

1. Pod 启动 → calico-cni 插件创建 veth pair → 分配 IP
2. calico-node 设置路由（BGP / IPIP）或封装规则
3. Pod 访问目标 Pod IP，Linux 路由表找到目标 Node IP
4. 使用 IPIP / VXLAN 封装（或直连） → 发给目标 Node
5. 解封装后通过 veth 交给目标 Pod 接收

------

## 🛠 四、Pod 不能通信时，如何一步步排查？（带命令）

| 步骤 | 检查点                      | 命令                                                         |
| ---- | --------------------------- | ------------------------------------------------------------ |
| 1️⃣    | 检查 Pod 是否 Ready         | `kubectl get pod -o wide`                                    |
| 2️⃣    | 测试网络连通性              | `kubectl exec pod-a -- ping <pod-b-ip>``kubectl exec pod-a -- curl <pod-b-ip>:port` |
| 3️⃣    | 判断是否跨节点              | `kubectl get pod -o wide` → 看 Node 列                       |
| 4️⃣    | 查看路由表和封装接口        | `ip route` 查看目标 Pod IP 路由`ip a` 查看是否有 tunl0（IPIP）或 caliXXX |
| 5️⃣    | 检查 NetworkPolicy 是否拦截 | `kubectl get networkpolicy -A``iptables -L -n -v` 查看规则   |
| 6️⃣    | 检查 Calico 运行状态        | `kubectl get pod -n kube-system -l k8s-app=calico-node -o wide``kubectl logs -n kube-system -l k8s-app=calico-node` |

------

### 🔧 实战命令示例：

```bash
# 1. 查看 Pod 的 IP 和所在节点
kubectl get pod -o wide

# 2. Pod 内 ping 另一个 Pod
kubectl exec pod-a -- ping 10.244.2.5

# 3. Pod 内 curl 服务
kubectl exec pod-a -- curl 10.244.2.5:80

# 4. 查看节点的路由表
ip route

# 5. 查看节点的网络接口（是否有 caliXXX、tunl0）
ip a

# 6. 查看所有网络策略
kubectl get networkpolicy -A

# 7. 查看 iptables 策略（注意看 cali 前缀的链）
iptables -L -n -v

# 8. 查看 calico-node 日志
kubectl logs -n kube-system -l k8s-app=calico-node
```

------

## ✅ 五、面试总结串词（推荐你直接背下来）

> Calico 是 Kubernetes 中常用的 CNI 插件，负责给 Pod 分配 IP 并建立通信路由。它默认使用 BGP 同步路由，也支持 IPIP 和 VXLAN 隧道，配合 Felix 设置 iptables，实现网络访问控制。
>  当 Pod 无法通信时，我会先确认 Pod 状态，然后测试网络连通性，检查是否跨节点，再看路由表和封装接口是否正常，最后排查是否有 NetworkPolicy 拦截，并查看 calico-node 是否运行正常。

