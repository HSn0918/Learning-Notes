#etcd #raft #demo

相关笔记：[[etcd-source]]

## 这是什么

`contrib/raftexample` 是 etcd 仓库里官方维护的「**raft 库最小可运行示例**」，几百行代码完整演示了上层应用如何接线 raft：propose、WAL、snapshot、HTTP transport、状态机（一个并发安全的 in-memory KV map）。它不是 etcd 的一部分，而是教你怎么用 `go.etcd.io/raft/v3` 这个独立库。

读完它你会理解：

- raft 库吐 `Ready` 给你之后，**上层到底要做哪些事**；
- `Tick / Propose / Step / Advance` 这几个动作分别在什么时机被调用；
- WAL 文件、Snapshotter 文件、raft Storage 三者怎么协作完成「重启恢复」；
- 多节点的 raft 消息是怎么走 HTTP `rafthttp` transport 传递的。

读 etcd 之前**强烈建议先读 raftexample**，否则一开始直奔 `server/etcdserver/server.go` 会被无数 etcd 自有的细节淹没。

## 在哪里

```bash
git clone https://github.com/etcd-io/etcd.git
cd etcd/contrib/raftexample
ls
# main.go  raft.go  kvstore.go  httpapi.go  listener.go  doc.go ...
```

或者直接在 GitHub 上看：<https://github.com/etcd-io/etcd/tree/main/contrib/raftexample>

## 文件布局（每个文件读什么）

| 文件 | 读什么 |
| --- | --- |
| `main.go` | 启动流程：参数解析、构造各个 channel、把 `kvstore` / `raftNode` / `httpKVAPI` 串起来。**先读这个文件**。 |
| `raft.go` | 核心。`raftNode` 结构体把 raft 库封装成「能跑的节点」，包含 WAL、Snapshotter、transport、和那个最重要的 `serveChannels()` Ready loop。 |
| `kvstore.go` | 应用层状态机：一个并发安全的 `map[string]string`。`Propose` 把 `kv` 序列化喂给 raft；`readCommits` 从 raft 拿 `commit` 出来 apply。 |
| `httpapi.go` | 一个最简陋的 HTTP API，把 `PUT /key` 翻译成 `kvstore.Propose`。 |
| `listener.go` | TLS / 端口监听辅助，可跳过。 |

## 关键函数：怎么读 `raft.go`

读 raftexample 的 `raft.go`，重点盯三件事：

### 1) 构造 raft.Node：是 `Start` 还是 `Restart`

```go
// raftexample/raft.go 中 startRaft 的逻辑（伪代码节选）
if oldwal {
    rc.node = raft.RestartNode(c)         // 从 WAL 恢复，不带 peers
} else {
    rc.node = raft.StartNode(c, rpeers)   // 全新集群
}
```

**关键判断**：本地有没有 WAL 决定了走「冷启动」还是「热恢复」。这条分支是 etcd 自己的 `bootstrap.go` 里同款逻辑的精简版。

### 2) `serveChannels()` 的 Ready loop

整个 raftexample 的灵魂。它在一个 goroutine 里 select：

- `rc.ticker.C`：每个 tick 调 `rc.node.Tick()`
- `rc.proposeC`：把 HTTP API 投递进来的 key/value 调 `rc.node.Propose(ctx, data)`
- `rc.confChangeC`：处理 ConfChange 提案
- **`rc.node.Ready()`**：核心分支。处理顺序严格按 etcd 的「先 WAL、再 send、再 apply、最后 Advance」：

```go
case rd := <-rc.node.Ready():
    rc.wal.Save(rd.HardState, rd.Entries)                  // 1) 持久化 HardState + Entries
    if !raft.IsEmptySnap(rd.Snapshot) {
        rc.saveSnap(rd.Snapshot)
        rc.raftStorage.ApplySnapshot(rd.Snapshot)
        rc.publishSnapshot(rd.Snapshot)
    }
    rc.raftStorage.Append(rd.Entries)                      // 2) 写入内存 raft storage
    rc.transport.Send(rc.processMessages(rd.Messages))     // 3) 发出去
    applyDoneC, ok := rc.publishEntries(rc.entriesToApply(rd.CommittedEntries))
    if !ok { rc.stop(); return }                           // 4) apply 到 kvstore
    rc.maybeTriggerSnapshot(applyDoneC)
    rc.node.Advance()                                       // 5) 最后通知 raft 推进
```

这五步顺序 = etcd 真实代码里的同款顺序，背下来面试稳了。

### 3) 重启恢复路径

`replayWAL()` 把 WAL 里的 entries 全部 `Append` 回 `raft.MemoryStorage`，然后 `RestartNode` 用这个 storage 起 raft。raft 库看到「我的 storage 里已经有日志了」就不会重复拉取，直接从最新点继续。

## 推荐的阅读顺序

1. **跑起来**：按官方 README 起 3 节点，curl 几个 key 看效果。
2. **读 `main.go`**：理解 `proposeC` / `confChangeC` / `commitC` / `errorC` 几条 channel 怎么把模块串起来。画一张图：HTTP → propose channel → raftNode → commit channel → kvstore。
3. **读 `kvstore.go`**：先读 `Propose`（编码 → 投 `proposeC`），再读 `readCommits`（从 raft 拿 commit → 解码 → 更新 map）。
4. **读 `raft.go` 的 `serveChannels()`**：对照上面那五步逐行看。
5. **读 `raft.go` 的 `replayWAL` / `loadSnapshot`**：理解重启恢复。

## 和 etcd 服务端代码的对应关系

| raftexample | etcd 真实代码 |
| --- | --- |
| `raftNode.serveChannels` 处理 `Ready` | `server/etcdserver/raft.go` 里 `raftNode.start()` 里的同款 select |
| `kvstore.readCommits` apply 已提交日志 | `EtcdServer.run` 收到 `s.r.apply()` 后 `applyAll()` |
| `httpKVAPI.Propose` 调用 `node.Propose` | `EtcdServer.processInternalRaftRequestOnce`（封装了 wait + 超时） |
| `replayWAL` | `server/etcdserver/bootstrap.go` 的 `bootstrap` |

读完 raftexample 后再去看 etcd，你会发现 etcd = raftexample + lease + mvcc + treeIndex + watchableStore + grpc——核心结构其实就这么点。

## 学到的东西如何对回 K8s

- 知道 `Ready` 五步以后，理解「K8s 的 resourceVersion 为什么是单调递增的」就简单了：每次 raft commit 推进 `revision`，apply 到 mvcc 后写进 boltdb，apiserver Get/List 把它装进 `ObjectMeta`。
- 看到 raftexample 也调 `rc.node.ReadIndex(...)` 实现线性一致读，再回去看 `EtcdServer.linearizableReadLoop` 就完全是一回事。
- ConfChange（增删节点）的处理流程，正是 K8s `kubeadm` 扩缩 etcd 集群时背后发生的事。
