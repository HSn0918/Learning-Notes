#kubernetes #cni #demo

相关笔记：[[cni-source]] | [[cni]] | [[calico]] | [[cilium]] | [[flannel]] | [[k8s-development-roadmap]] | [[network-model]]

## 概述

本 demo 用 **100 行 bash** 实现一个符合 CNI 0.4.0 规范的 bridge 插件 —— 跟 `containernetworking-plugins` 仓库的真实 bridge 插件做完全一样的事：在宿主机起 `cni0` 网桥，给每个 "Pod"（netns）分一个 IP + veth pair + 默认路由。

学习意义：CNI 是 K8s 网络的"地基"，但所有正经实现（Calico / Cilium / Flannel）都是 Go 写的、几千行起，新手读源码很容易被框架细节淹没。这个 100 行 bash 版剥掉一切框架，把 **CNI 的协议契约**（env + stdin JSON + stdout JSON）和**最小网络模型**（veth + bridge + IPAM）一次说清。读完它再去看真实插件源码，就能直接锁定"框架代码 vs 网络逻辑"的分界线。

配套源码 / 命令 / 跑测步骤详见 [README](./README.md)；CNI 协议本身的源码导读见 [[cni-source]]。

## 设计要点

1. **bash 而不是 Go**：CNI 插件本质是"读 stdin、写 stdout、调 ip 命令"的程序，bash 把每行的"意图"直接暴露给你。Go 版会被 `libcni` / `netlink` 包遮住一半视野。
2. **`run-in-docker.sh` 兜底 Mac**：Mac 没有 `netns` / `iptables` / `ip` 命令；用 `docker run --privileged ubuntu` 借 Mac 的 Linux VM kernel（Docker Desktop 就是个 Linux VM）跑全套网络工具，Mac 用户能"零依赖"复现。
3. **故意不做 IPAM 回收**：真实 IPAM（host-local）会维护 `/var/lib/cni/networks/<network>/<ip>` 文件做分配/回收，本 demo 简化为单调递增计数器（`/tmp/learning-bridge.next`），让你专注看"协议怎么对接"而不是"IPAM 怎么实现"。
4. **故意做 SNAT**：让 Pod 能 ping 外网，验证"宿主机网络到 Pod 网络"的端到端链路。真实 K8s 里 SNAT 通常由 kube-proxy 或 Calico/Cilium 自己接管，本 demo 保持自洽。
5. **不做 CHECK 实现**：CHECK 是 CNI 1.0 才强制的可选命令，本 demo 直接返回成功；真实插件应该把 ADD 写入的状态再读一遍验证。

## walkthrough：一次 ADD 调用做的 6 件事

```
kubelet (或本 demo 的 run-in-docker.sh)
  ├─ 1) ip netns add pod-xxx     <- 给 Pod 建 sandbox netns (本来是 pause 容器干)
  └─ 2) 调插件:
         CNI_COMMAND=ADD
         CNI_CONTAINERID=xxx
         CNI_NETNS=/run/netns/pod-xxx
         CNI_IFNAME=eth0
         stdin: {"subnet":"10.244.0.0/24","gateway":"10.244.0.1","bridge":"cni0"}
         ↓
         learning-bridge:
            a) ensure_bridge:
                  ip link add cni0 type bridge
                  ip addr add 10.244.0.1/24 dev cni0
                  ip link set cni0 up
                  sysctl -w net.ipv4.ip_forward=1
                  iptables -t nat -A POSTROUTING -s 10.244.0.0/24 ! -o cni0 -j MASQUERADE
            b) IPAM next_ip -> 10.244.0.3/24
            c) veth pair:
                  ip link add veth<8位ID> type veth peer name ctr<8位ID>
                  ip link set veth<8位ID> master cni0          <- host 端入网桥
                  ip link set veth<8位ID> up
                  ip link set ctr<8位ID> netns /run/netns/pod-xxx
                  ip -n pod-xxx link set ctr<8位ID> name eth0  <- container 端重命名
                  ip -n pod-xxx addr add 10.244.0.3/24 dev eth0
                  ip -n pod-xxx link set eth0 up
                  ip -n pod-xxx link set lo up
                  ip -n pod-xxx route add default via 10.244.0.1
            d) 输出 JSON:
                  {"cniVersion":"0.4.0",
                   "interfaces":[{"name":"cni0",...},{"name":"eth0","sandbox":"/run/netns/pod-xxx",...}],
                   "ips":[{"address":"10.244.0.3/24","gateway":"10.244.0.1",...}],
                   "routes":[{"dst":"0.0.0.0/0","gw":"10.244.0.1"}]}
```

第 c 步是 CNI 网络模型的核心：**veth pair 是一根虚拟"双绞线"，两端分别在 host 与 container 命名空间**，host 端入网桥享受网桥转发，container 端就是 Pod 看到的 `eth0`。

## 跟生产 CNI 实现的对照

| 关键差异 | 本 demo | 生产 (Calico/Cilium/Flannel) |
| :--- | :--- | :--- |
| 实现语言 | bash | Go + netlink/eBPF |
| IPAM | 计数器 | host-local / dhcp / etcd / 自家中心化 IPAM |
| 跨节点连通 | 无 | VXLAN(Flannel) / BGP(Calico) / eBPF(Cilium) |
| NetworkPolicy | 无 | iptables 链(Calico) / eBPF 程序(Cilium) |
| Service 转发 | 无（依赖 kube-proxy） | 可选自带，Cilium 可替换 kube-proxy |
| 多网卡 | 无 | Multus 协调多 CNI |

## 在 [[k8s-development-roadmap]] / [[progress]] 中的位置

- **K8s 路线图第 8 主题（CNI）的「衡量标准」**："能在一台机器上手写一个最简 CNI 插件实现 ADD/DEL" → 这个 demo 就是产出物。
- **progress.md Week 6**："手写最简 bridge CNI（shell + ip 命令）" → 直接 `./run-in-docker.sh`，看完读 [[cni-source]] 对照真实源码。

## 局限

- **不能加进真 kubelet 用**：缺 `.conflist` 配置、缺 hostport / portmap 配套、CHECK 不实现，kubelet CNI 校验会失败。要加 kubelet 用最少还要：把 IPAM 改用 host-local、加 portmap 链插件、写 `.conflist`。
- **重启 docker 后 IPAM 计数重置**：是有意的（学习用），生产要持久化。
- **没做 hairpin/promisc**：同 Pod 自己访问自己 Service 的场景在生产里要打 hairpin，本 demo 不演示。
