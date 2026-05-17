# cni-bridge

> **状态**：✅ bash 脚本语法校验通过；Mac 跑需 docker（`./run-in-docker.sh` 一键拉 ubuntu 容器） · 详见 [demos 验证总表](../README.md)

100 行 bash 实现的 **CNI 0.4.0 bridge 插件**，配 docker 一键演示完整 `ADD` / `DEL` 协议、veth pair、IPAM。Mac 用户用 `./run-in-docker.sh` 一键跑。

配套阅读：
- [[demo-cni-bridge]] —— 本目录的笔记入口与详细 walkthrough
- [[cni-source]] —— CNI 协议 + containernetworking 仓库源码导读（含真实 bridge 插件源码逐段拆解）
- [[cni]] —— 概念层 + 插件选型对比

## 它能让你看到什么

跑完 `./run-in-docker.sh` 输出大致是：

```
=== ADD pod-aaaaaaaa1111 ===
{
  "cniVersion": "0.4.0",
  "interfaces": [
    {"name": "cni0", "mac": "...:..:..:..:..:.."},
    {"name": "eth0", "mac": "...:..:..:..:..:..", "sandbox": "/run/netns/pod-aaaaaaaa1111"}
  ],
  "ips": [
    {"version": "4", "address": "10.244.0.3/24", "gateway": "10.244.0.1", "interface": 1}
  ],
  ...
}

=== 互 ping (pod-a -> pod-b) ===
PING 10.244.0.4 (10.244.0.4): 56 data bytes
64 bytes from 10.244.0.4: icmp_seq=0 ttl=64 time=0.123 ms
64 bytes from 10.244.0.4: icmp_seq=1 ttl=64 time=0.045 ms
64 bytes from 10.244.0.4: icmp_seq=2 ttl=64 time=0.041 ms
```

**关键观察**：
1. 你写了一个 bash 脚本，它符合 CNI 规范，环境变量传命令、stdin 传 JSON、stdout 返回 JSON
2. 两个独立 netns（"假 Pod"）通过你创建的 `cni0` 网桥互通
3. `DEL` 之后 veth 自动清理

把 `kubelet` 换成本脚本的 `run-in-docker.sh`，把 `/run/netns/...` 换成 kubelet 创建的真 sandbox netns —— **就是真实 Kubernetes 的 CNI 流程**。

## 跑起来（Mac）

```bash
cd learning-plan/demos/cni-bridge
./run-in-docker.sh
```

需要 docker。脚本里 `--privileged` 借 Mac 宿主机 kernel 跑 netns / iptables / ip 命令。

## 跑起来（Linux）

不用 docker，root 直接跑：

```bash
NETCONF='{"cniVersion":"0.4.0","name":"learn","type":"learning-bridge","subnet":"10.244.0.0/24","gateway":"10.244.0.1","bridge":"cni0"}'

ip netns add pod-1
CNI_COMMAND=ADD CNI_CONTAINERID=abc1 CNI_NETNS=/run/netns/pod-1 CNI_IFNAME=eth0 \
  ./learning-bridge <<< "$NETCONF"

ip netns exec pod-1 ip a
```

## 协议 cheatsheet（背下来）

| 输入 | 媒介 |
| :--- | :--- |
| 命令（ADD/DEL/CHECK/VERSION） | env `CNI_COMMAND` |
| 容器 ID | env `CNI_CONTAINERID` |
| 容器 netns 路径 | env `CNI_NETNS` |
| 容器内网卡名 | env `CNI_IFNAME`（kubelet 总传 `eth0`） |
| 插件搜索路径 | env `CNI_PATH` |
| 网络配置 | **stdin JSON** |

| 输出 | 媒介 |
| :--- | :--- |
| 成功结果 | stdout JSON（`interfaces` / `ips` / `routes` / `dns`） |
| 错误 | stdout JSON `{"code":N,"msg":"..."}` + exit non-zero |

| 默认安装路径 | 用途 |
| :--- | :--- |
| `/opt/cni/bin/` | 插件二进制（kubelet `--cni-bin-dir`） |
| `/etc/cni/net.d/` | `.conflist` 配置（kubelet `--cni-conf-dir`） |

## walkthrough 7 步

```
1) docker run --privileged ubuntu  -> 装 iproute2/iptables/jq
2) ip netns add pod-aaaaaaaa1111   -> 模拟 kubelet 给 Pod 建 sandbox netns
3) 调插件 (env+stdin) ADD          -> learning-bridge 干 3 件事:
       a) ensure_bridge: 没 cni0 就建, 设网关 IP, 开 ip_forward+SNAT
       b) IPAM: /tmp/learning-bridge.next 计数, 取下一个 IP
       c) veth pair: host 端入网桥, container 端进 netns 改名 eth0 + 配 IP + 默认路由
4) 同样 ADD pod-bbbbbbbb2222        -> 复用 cni0, 拿下一个 IP
5) ip netns exec pod-aaa ping <pod-b 的 IP>
        -> 走 eth0 -> veth -> cni0 网桥转 -> 对端 veth -> eth0
6) DEL                              -> ip link del veth<id> (netns 端自动消失)
7) ip netns del pod-...             -> 清理 sandbox
```

## 它和 containernetworking-plugins 的差距

| 真实 bridge 插件 | 本 demo |
| :--- | :--- |
| Go 写, 用 libcni / netlink 库 | 100 行 bash + ip 命令 |
| IPAM 走独立插件 (host-local / dhcp / static) | 用 /tmp 状态文件计数 |
| 完整 IPv6 / hairpin / promisc 配置 | 只 IPv4 |
| 严格的 CHECK 实现 | 永远返回成功 |
| Error code 按 SPEC 分类 | 出错就 exit 1 |
| 网络策略 / NetworkPolicy | 无 |
| 多 subnet / 多网卡 | 无 |
| Hostport / portmap 链 | 无 |

补齐这些差距 = 你写出了一个生产可用的 CNI 插件。

## 跟 Calico / Cilium / Flannel 的关系

它们都遵守同一个 CNI 协议，区别在"网络模型"：

| 插件 | 大致做法 |
| :--- | :--- |
| **bridge (本 demo)** | 同节点 cni0 网桥, 跨节点要外层路由（默认不做） |
| **Flannel** | bridge + VXLAN: 节点间封一层 UDP overlay |
| **Calico** | 不用网桥, 每个 Pod 单独 veth, 用 BGP 在节点间播路由 |
| **Cilium** | 不用 iptables, 用 eBPF 程序挂 tc/xdp hook |

读懂本 demo 100 行 bash → 你就读懂了所有 CNI 插件的"骨架"，剩下的是不同流派的网络模型。

## 清理

`./run-in-docker.sh` 跑完 docker 容器自动销毁，不污染 Mac 宿主机。

如果你直接在 Linux 上跑过：

```bash
ip link del cni0
iptables -t nat -D POSTROUTING -s 10.244.0.0/24 ! -o cni0 -j MASQUERADE
rm -f /tmp/learning-bridge.next /tmp/learning-bridge.log
ip -all netns delete
```
