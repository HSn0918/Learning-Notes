# etcd-client-demo

用 `go.etcd.io/etcd/client/v3` 跑一遍 Put / Get / Watch / Lease，把 etcd 的 **revision** 概念直接打印出来——这就是 K8s `resourceVersion` 的底层物。配套笔记：[[etcd-source]]。

## 1. 起一个本地 etcd

```bash
docker run -d -p 2379:2379 --name etcd-demo \
  quay.io/coreos/etcd:v3.5.18 \
  etcd \
  --advertise-client-urls=http://0.0.0.0:2379 \
  --listen-client-urls=http://0.0.0.0:2379
```

确认能连：

```bash
docker exec etcd-demo etcdctl endpoint status --write-out=table
```

## 2. 跑 demo

```bash
cd learning-plan/demos/raftexample-walkthrough/etcd-client-demo
go mod tidy   # 自己跑，本仓库不提交 go.sum
go run .
```

## 3. 预期输出（每一行都对应笔记里的一个概念）

```
[start] current etcd revision = 2, watch from rev=3
[put]   #0 value=v1 -> revision=3          ← 每次 Put 分配一个全局递增的 revision
[put]   #1 value=v2 -> revision=4
[put]   #2 value=v3 -> revision=5
[watch] rev=3 type=PUT key=/learning-notes/etcd-demo/foo value=v1
[watch] rev=4 type=PUT key=/learning-notes/etcd-demo/foo value=v2
[watch] rev=5 type=PUT key=/learning-notes/etcd-demo/foo value=v3
[get]   key=/learning-notes/etcd-demo/foo value=v3 create=3 mod=5 version=3 (lease=0)
                                              ↑ CreateRevision 是首次写入的 rev
                                                     ↑ ModRevision 是最近一次写入的 rev
                                                            ↑ Version 是该 key 的修改次数
[get@rev=3] value=v1                        ← MVCC 让我们能读历史 revision 的值
[lease] id=<id> ttl=5s key=/learning-notes/etcd-demo/lease-bound (5s 后被自动 revoke)
[watch] rev=N type=PUT  key=.../lease-bound  value=ephemeral
[watch] rev=M type=DELETE key=.../lease-bound (lease 过期触发删除)
[lease] after TTL: /learning-notes/etcd-demo/lease-bound 还存在? false
```

## 4. 看到了什么、对应源码哪里

| 看到的现象 | 对应源码 |
| --- | --- |
| 每次 `Put` 返回一个递增的 `revision` | `server/storage/mvcc/kvstore_txn.go:196` 的 `storeTxnWrite.put`，事务 End 时 `currentRev++` |
| `CreateRevision != ModRevision` | `put` 沿用之前的 `created.Main` 作为 `CreateRevision` |
| `Watch(WithRev(N))` 能拿到 N 之后的事件 | 同 K8s `staging/.../etcd3/watcher.go:381` 的 `clientv3.WithRev(initialRev+1)` |
| `Get(WithRev(oldRev))` 能读出历史值 | MVCC：boltdb 物理 key 是 revision，老 revision 的 KeyValue 仍在 |
| Lease 过期后 key 自动消失（一条 DELETE 事件） | `server/lease/lessor.go`，Leader 发 `LeaseRevoke` 提案 |

## 5. 清理

```bash
docker rm -f etcd-demo
```
