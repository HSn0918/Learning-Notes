#etcd #raft #demo

相关笔记：[[etcd-source]]

## 这个目录是什么

围绕 [[etcd-source]] 里的概念做的两个配套教学件，**不是一个完整 raft 集群实现**——etcd 的 raft 是一个独立库，自己拼一个完整集群代码量太大；这里换两条路把核心概念落到「能跑」或「能读」的程度：

| 文件 | 目的 | 怎么用 |
| --- | --- | --- |
| `etcd-client-demo/` | 用 clientv3 连真实 etcd，把 **revision / MVCC / Watch / Lease** 概念**跑出来**给你看 | `go run .`，输出每行都对应笔记里的一个概念 |
| `raftexample-notes.md` | etcd 官方 `contrib/raftexample` 项目的**阅读指南** | 配合 etcd 源码读，理解 raft 库的接线方式 |

## 推荐学习顺序

1. 先读 [[etcd-source]]，把整体架构、Ready loop、MVCC、Watch 模型过一遍。
2. 跑 `etcd-client-demo/`，亲眼看到 revision 递增、Watch 事件流、Lease 自动过期。这让 [[etcd-source]] 里的「写入分配 revision」「resourceVersion 就是 revision」从抽象变成具体。
3. 按 [[raftexample-notes]] 的导读顺序读 etcd 仓库里 `contrib/raftexample` 的 4 个核心文件。
4. 回过头再读 etcd `server/etcdserver/server.go` 的 `run()` 和 `raft.go`，会发现就是 raftexample + 额外的 lease/mvcc/grpc。

## 为什么不写一个手搓的 raft 集群

- raft 算法本身（选举、日志复制、安全性证明）在面试和工程里几乎从不要求重新实现。
- 真正高频被问的是「**raft 库 / 应用层的接线方式**」：Ready 五步、WAL 与 snapshot 协作、ReadIndex。
- 这两点用 raftexample 阅读 + clientv3 跑通完全可以达到。要更深入直接 fork raftexample 改即可。

## 笔记里的「手写简化复现」在哪里

[[etcd-source]] 「Raft 模块：纯状态机设计」章节有一段 **手写简化复现：Ready loop pattern** 的 60 行 Go 代码，定义了 `Entry / Message / Ready / Node` 和一个最小的 `run()` 事件循环，演示「Propose → Ready → 上层处理（WAL→Send→Apply）→ Advance」的握手节奏。它不依赖任何外部库，直接看代码就能理解 raft 库与上层的协议。

读这段简化代码 + 笔记里 `vendor/go.etcd.io/raft/v3/node.go:343-454` 的真实 `node.run`，二者并列，理解整个 Ready 协议无障碍。
