#go #runtime #map

相关笔记：[[go-map-source]] | [[go-slice-source]] | [[go-gc-source]] | [[map-internals]]

# Go Map 新旧实现对比

## 概述

Go map 的语言语义没有因为底层实现切换而改变，但源码结构和性能模型已经发生明显变化：
- Go 1.23 及更早的经典模型：`hmap + bmap + overflow bucket + oldbuckets evacuation`。
- Go 1.24+ 当前模型：`internal/runtime/maps`，Swiss Table 风格的 `Map + Table + Group + Control word + Directory`。

这篇文档的目标是把“旧 map”和“新 map”分清楚，避免面试和排障时拿旧源码解释当前 Go 版本。

当前本机源码版本：

```text
go version go1.26.1 darwin/arm64
GOROOT=/opt/homebrew/Cellar/go/1.26.1/libexec
```

## 版本边界

| 版本 | 实现主线 | 源码入口 |
|------|----------|----------|
| Go 1.23 及更早 | bucket hash table | `runtime/map.go`, `runtime/map_fast*.go` |
| Go 1.24+ | Swiss Table + extendible hashing | `internal/runtime/maps/*`, `runtime/map.go` wrappers |

注意：如果读的是老文章，里面大概率会讲 `hmap.B`、`buckets`、`oldbuckets`、`nevacuate`、`overflow`。这些概念对理解历史实现仍然有价值，但不能直接套到 Go 1.24+ 源码。

## 旧实现：hmap / bmap / overflow

### 结构模型

旧实现可以抽象成：

```go
type hmap struct {
    count     int
    flags     uint8
    B         uint8
    noverflow uint16
    hash0     uint32
    buckets    unsafe.Pointer
    oldbuckets unsafe.Pointer
    nevacuate  uintptr
    extra      *mapextra
}

type bmap struct {
    tophash [8]uint8
    keys    [8]keytype
    values  [8]valuetype
    overflow *bmap
}
```

这是历史模型的简化结构，不是 Go 1.26 当前源码中的结构。

### 查找

旧 map 查找主线：

```text
hash(key)
bucket index = hash & (2^B - 1)
scan bucket tophash[8]
compare candidate keys
follow overflow chain if needed
if growing, also check oldbucket/evacuated state
```

关键点：
- 每个 bucket 8 个 key/value。
- `tophash` 保存 hash 高位，先过滤再比较 key。
- 冲突多时会挂 overflow bucket。
- overflow chain 变长会损害局部性和尾延迟。

### 扩容

旧 map 有两类 grow：
- bigger grow：bucket 数量翻倍。
- same-size grow：bucket 数量不变，主要清理 overflow。

为了避免一次搬完所有 bucket，旧 map 使用渐进式 evacuation：
- `oldbuckets` 指向旧 bucket 数组。
- `nevacuate` 记录迁移进度。
- 每次 map 写入时顺手搬迁一部分 bucket。

### 旧实现的典型问题

| 问题 | 原因 |
|------|------|
| overflow chain 长 | hash 冲突或负载增长导致 overflow bucket 多 |
| 局部性一般 | bucket 和 overflow bucket 可能分散 |
| grow 状态复杂 | 需要同时处理 buckets、oldbuckets、evacuated marker |
| 删除不缩容 | delete 不会主动把 map 缩小 |

## 新实现：Swiss Table / Group / Directory

### 结构模型

当前 Go 1.26.1：

```go
type Map struct {
    used       uint64
    seed       uintptr
    dirPtr     unsafe.Pointer
    dirLen     int
    globalDepth uint8
    globalShift uint8
    writing    uint8
    tombstonePossible bool
    clearSeq   uint64
}

type table struct {
    used       uint16
    capacity   uint16
    growthLeft uint16
    localDepth uint8
    index      int
    groups     groupsReference
}
```

核心变化：
- 不再以 overflow bucket 作为主要冲突处理方式。
- table 使用 open addressing + probing。
- 每个 group 有 8 个 slot 和 control word。
- 顶层 Map 可以有多个 table，通过 directory 选择。

### 查找

新 map 查找主线：

```text
hash(key, seed)
split hash into H1 and H2
directory selects table by H1 high bits
table chooses initial group
quadratic probing over groups
control word matches 8 H2 bytes in parallel
real key equality for candidates
stop at empty slot
```

### 扩容

新 map 面临的问题：Swiss Table table grow 时，probe sequence 依赖 group count，扩容要重排整个 table。

Go 的解决方案：
- 小 table 可以整体替换成双倍容量 table。
- 单个 table 最大容量受 `maxTableCapacity` 限制。
- 超过限制后 table split。
- 顶层 directory 使用 extendible hashing 选择 table。
- 这样单次 grow 控制在局部 table 内。

### 删除

新 map 使用 tombstone：
- probe 需要继续的位置不能直接变 empty。
- insert 优先复用 tombstone。
- rehash/grow 时清理 tombstone。

## 核心差异表

| 维度 | 旧 map | 新 map |
|------|--------|--------|
| 主结构 | `hmap/bmap` | `maps.Map/table/group` |
| 冲突处理 | bucket + overflow chain | open addressing + quadratic probing |
| 过滤 hash | `tophash[8]` | control word 中的 H2 |
| group/bucket 宽度 | bucket 8 slots | group 8 slots |
| 局部性 | overflow 可能分散 | group/control word 更紧凑 |
| 扩容 | oldbuckets 渐进 evacuation | table grow/split + directory |
| 删除 | bucket slot 状态 | tombstone + rehash 清理 |
| 迭代 | bucket/oldbucket 状态复杂 | old table + new table lookup 维持语义 |
| 源码入口 | `runtime/map.go` | `internal/runtime/maps/*` |

## 哪些语义没变

### map 仍然不是并发安全

普通 map 并发读写仍然是 data race，并可能 panic：

```text
fatal error: concurrent map writes
fatal error: concurrent map read and map write
```

新实现的 `writing` 字段只是检测部分错误，不是同步原语。

### 迭代顺序仍然不保证

不要依赖 map range 顺序。即使当前某次运行看起来稳定，也不属于语言保证。

### delete 仍然不自动缩容

大量 delete 后，map 仍可能保留底层容量和对象引用。需要手动重建 map 或设计缓存淘汰。

### key 仍然必须可比较

slice、map、func 仍然不能作为 map key。interface key 的动态值也必须可比较，否则运行时 panic。

## 对工程实践的影响

### 老文章还能不能看

能看，但要带版本标签：
- 想理解 Go map 历史和传统面试题，旧 `hmap/bmap` 文章仍有价值。
- 想解释当前 Go 1.24+ 源码，就必须读 `internal/runtime/maps`。
- 如果面试官仍问 `hmap`，可以先回答“旧实现是这样”，再补一句“Go 1.24+ 已切到 Swiss Table 主线”。

### unsafe/linkname 风险变高

很多包曾经通过 `go:linkname` 依赖 runtime map 内部函数。map 实现迁移后，runtime 仍保留一部分 wrapper 兼容，但任何依赖内部布局的 unsafe 代码都非常脆弱。

排查：

```bash
rg -n 'go:linkname|runtime\\.map|hmap|bmap|unsafe\\.Pointer' --glob '*.go'
```

### benchmark 需要重跑

新 map 通常改善局部性和查找路径，但具体业务要看：
- key 类型：int/string/struct/interface。
- value 大小和是否含指针。
- map 大小。
- insert/delete 比例。
- 是否大量迭代。
- 是否存在 churn 和 tombstone。

不要把运行时实现变化直接等同于所有业务场景都会变快。

## 面试要点

### Q: Go map 新旧实现最大的区别是什么？

A: 旧实现是 `hmap/bmap/overflow bucket`，冲突主要通过 bucket 链解决，并用 `oldbuckets/nevacuate` 渐进迁移。Go 1.24+ 新实现是 Swiss Table 风格，用 `Map/Table/Group/Control word/Directory`，group 内用 H2 control byte 快速过滤，table 层用 open addressing 和 probing，顶层用 extendible hashing 控制 grow 粒度。

### Q: 新 map 为什么比旧 map 更关注 control word？

A: control word 把 8 个 slot 的状态和 H2 hash 放在紧凑元数据中。查找时可以一次比较一个 group 的多个 H2，减少逐 slot key 比较和随机访问，从而改善局部性。

### Q: 旧 map 的 overflow bucket 问题是什么？

A: 冲突多时 overflow chain 变长，查找要追指针，局部性变差，尾延迟上升。same-size grow 可以清理 overflow，但实现状态更复杂。

### Q: 新 map 为什么还需要多 table？

A: 单个 Swiss Table 扩容需要重排整个 table。Go 用多个 table 加 directory，把大 map 拆成较小 table，单次只 grow 或 split 一个 table，控制延迟。

### Q: 新 map 是否改变了 map 的并发安全语义？

A: 没有。普通 map 仍然不能并发读写。新实现有 `writing` 检测并发写，但那只是 fatal 检测，不是同步。工程上仍要用锁、单 owner goroutine 或并发容器。

### Q: 新 map 是否会自动缩容？

A: 不会。delete 不保证释放底层 table 或降低容量。长生命周期大 map 删除大量 key 后，仍建议重建 map 或做缓存淘汰。

### Q: 面试官问 hmap 怎么办？

A: 先说明版本边界：`hmap/bmap` 是 Go 1.23 及更早的经典实现；再按旧模型回答 bucket、tophash、overflow、evacuation；最后补充当前 Go 1.24+ 已迁移到 `internal/runtime/maps` 的 Swiss Table 实现。
