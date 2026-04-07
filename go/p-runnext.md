#go #gmp调度模型

相关笔记：[[gmp-model]] | [[context]]

## P.runnext 机制

`P.runnext` 是 Go 调度器中的一个重要优化：每个 P 上有一个特殊的 `runnext` 字段，存储**下一个优先执行的 goroutine**。它的优先级高于 P 的本地队列和全局队列。

### 调度优先级

```
P.runnext > P.localrunq > globalrunq
```

---

## 问题引入

以下代码的输出是什么？

```go
package main

import (
    "fmt"
    "runtime"
    "sync"
)

func main() {
    runtime.GOMAXPROCS(1) // 单线程
    wg := &sync.WaitGroup{}
    wg.Add(3)

    go func() { fmt.Println("A"); wg.Done() }()  // G1
    go func() { fmt.Println("B"); wg.Done() }()  // G2
    go func() { fmt.Println("C"); wg.Done() }()  // G3

    wg.Wait()
}
```

输出（稳定）：

```
C
A
B
```

为什么不是 A、B、C 的顺序？这涉及 `P.runnext` 的工作原理。

---

## go 关键字的底层实现

`go func()` 在编译期被转换为 `runtime.newproc`：

```go
// runtime/proc.go
func newproc1(fn *funcval, argp unsafe.Pointer, narg int32,
    callergp *g, callerpc uintptr) {

    _g_ := getg()
    _p_ := _g_.m.p.ptr()
    newg := gfget(_p_)

    casgstatus(newg, _Gdead, _Grunnable)
    newg.goid = int64(_p_.goidcache)

    // 关键：第三个参数 next=true，放入 P.runnext
    runqput(_p_, newg, true)

    if atomic.Load(&sched.npidle) != 0 && atomic.Load(&sched.nmspinning) == 0 && mainStarted {
        wakep()
    }
    releasem(_g_.m)
}
```

注意 `runqput(_p_, newg, true)` 的第三个参数 `next=true`，表示新 goroutine 会被放入 `P.runnext`。

---

## runqput 源码分析

```go
// runtime/proc.go
func runqput(_p_ *p, gp *g, next bool) {
    if next {
    retryNext:
        oldnext := _p_.runnext
        // CAS 将新 G 放入 runnext，与旧值交换
        if !_p_.runnext.cas(oldnext, guintptr(unsafe.Pointer(gp))) {
            goto retryNext
        }
        if oldnext == 0 {
            return // runnext 之前为空，直接返回
        }
        // runnext 之前不为空，被挤出的旧 G 需要放入本地队列
        gp = oldnext.ptr()
    }

    // 将 gp 放入本地队列尾部
    // 如果 next=false，gp 是新创建的 goroutine
    // 如果 next=true，gp 是从 runnext 被挤出的旧 goroutine
retry:
    h := atomic.LoadAcq(&_p_.runqhead)
    t := _p_.runqtail
    if t-h < uint32(len(_p_.runq)) {
        _p_.runq[t%uint32(len(_p_.runq))].set(gp)
        atomic.StoreRel(&_p_.runqtail, t+1)
        return
    }
    // 本地队列已满，放入全局队列
    if runqputslow(_p_, gp, h, t) {
        return
    }
    goto retry
}
```

核心逻辑：
1. 新 goroutine 通过 CAS 操作放入 `P.runnext`
2. 如果 `runnext` 已有旧值，旧值被**挤出**到本地队列尾部
3. 本地队列满时，溢出到全局队列

---

## 执行顺序推导

```mermaid
flowchart TD
    subgraph "Step 1: go func A (创建 G1)"
        S1_RN["runnext: G1"]
        S1_LQ["localrunq: (empty)"]
    end

    subgraph "Step 2: go func B (创建 G2)"
        S2_RN["runnext: G2"]
        S2_LQ["localrunq: G1 ← (G1 被挤出)"]
    end

    subgraph "Step 3: go func C (创建 G3)"
        S3_RN["runnext: G3"]
        S3_LQ["localrunq: G1, G2 ← (G2 被挤出)"]
    end

    S1_RN --> S2_RN
    S2_RN --> S3_RN
```

| 步骤 | 操作 | runnext | 本地队列 |
|------|------|---------|----------|
| 1 | 创建 G1（打印 A） | G1 | [] |
| 2 | 创建 G2（打印 B） | G2 | [G1] （G1 被挤出） |
| 3 | 创建 G3（打印 C） | G3 | [G1, G2] （G2 被挤出） |

### 调度顺序

调度器获取 G 的顺序（`runqget` 函数）：

1. **先取 `runnext`** → G3（打印 C）
2. **再从本地队列头部取** → G1（打印 A）
3. **继续取** → G2（打印 B）

所以输出是 `C → A → B`。

---

## runqget 源码

```go
func runqget(_p_ *p) (gp *g, inheritTime bool) {
    // 优先从 runnext 获取
    for {
        next := _p_.runnext
        if next == 0 {
            break
        }
        if _p_.runnext.cas(next, 0) {
            return next.ptr(), true
        }
    }
    // runnext 为空，从本地队列 FIFO 获取
    for {
        h := atomic.LoadAcq(&_p_.runqhead)
        t := _p_.runqtail
        if t == h {
            return nil, false
        }
        gp := _p_.runq[h%uint32(len(_p_.runq))].ptr()
        if atomic.CasRel(&_p_.runqhead, h, h+1) {
            return gp, false
        }
    }
}
```

---

## Go 1.13 的特殊情况

在 Go 1.13 中，以下代码输出 `A B C` 而非 `C A B`：

```go
func main() {
    runtime.GOMAXPROCS(1)
    go func() { fmt.Println("A") }()
    go func() { fmt.Println("B") }()
    go func() { fmt.Println("C") }()
    time.Sleep(time.Second)
}
```

原因：Go 1.13 中 `time.Sleep` 会隐式启动一个 `timerproc` goroutine（用于监控 timer bucket），这个额外的 goroutine 会影响 `runnext` 的状态，改变最终的执行顺序。

Go 1.14+ 将 timer 检查移入了 `schedule()` 函数，不再启动额外的 goroutine，因此该问题不再存在。

---

## 哪些场景会使用 P.runnext

以下情况 goroutine 会被放入 `P.runnext`（获得"插队"权）：

| 场景 | 说明 |
|------|------|
| `go func()` | 新创建的 goroutine |
| `goready()` | 被唤醒的 goroutine（如 channel 接收方） |
| `newproc` | runtime 内部创建 goroutine |

设计意图：**刚创建或刚被唤醒的 goroutine 通常有更高的时间局部性**，优先执行可以减少调度延迟、提高缓存命中率。

---

## 总结

```mermaid
graph TD
    NEW["新建/唤醒 goroutine"] --> RN["P.runnext\n(最高优先级，只存 1 个)"]
    RN -->|被挤出| LQ["P.localrunq\n(环形队列，容量 256)"]
    LQ -->|队列满| GQ["globalrunq\n(全局队列，需加锁)"]

    SCH["调度器 schedule()"] -->|1. 先取| RN
    SCH -->|2. 再取| LQ
    SCH -->|3. 每61次检查| GQ
    SCH -->|4. 最后| WS["Work Stealing\n(偷其他 P 的一半)"]
```
