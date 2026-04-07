#kubernetes #cni #weave

相关笔记：[[cni]] | [[network-model]] | [[flannel]]

## Weave Net 概述

Weave Net 由 Weaveworks 开发，特点是部署极简且内置加密通信能力。

## 两种数据通道

| 模式 | 名称 | 原理 | 性能 |
| --- | --- | --- | --- |
| Fast Datapath | **fastdp** | 使用 Linux 内核的 Open vSwitch datapath（内核态转发） | 高 |
| 用户态转发 | **sleeve** | 当 fastdp 不可用时回退，通过用户态进程封装/转发 | 低 |

Weave 会自动检测环境，优先使用 fastdp，只有在内核不支持时才回退到 sleeve 模式。

## 加密通信

Weave Net 内置 IPsec 加密（使用 NaCl 库），只需在安装时加一个环境变量即可启用：

```bash
# 安装 Weave Net 并启用加密
kubectl apply -f "https://cloud.weave.works/k8s/net?k8s-version=$(kubectl version | base64 | tr -d '\n')&password-secret=weave-passwd"
```

加密对所有 Pod 间流量透明生效，不需要应用层改造。

## 适用场景

- 中小集群，需要开箱即用的加密通信
- 多云/混合云环境，节点间需要加密传输
- 追求部署简便（单条命令即可安装）
- 不需要复杂的网络策略（Weave 支持基础 NetworkPolicy）

## 注意事项

Weaveworks 已经关闭，Weave Net 长期维护存疑。新项目建议优先考虑 Calico 或 Cilium，如果需要加密通信可以使用它们的 WireGuard 集成。

## 面试要点

**Q: Weave Net 的 fastdp 和 sleeve 模式有什么区别？**

A: fastdp 使用 Linux 内核的 Open vSwitch datapath 在内核态转发数据包，性能高；sleeve 是用户态转发的回退方案，性能较低。Weave 自动检测环境，优先使用 fastdp。

**Q: Weave Net 的加密如何工作？**

A: Weave Net 内置 NaCl 加密库，安装时指定密码即可启用 IPsec 加密，对所有 Pod 间流量透明生效，无需应用层改造。
