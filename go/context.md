#go #context

相关笔记：[[gmp-model]] | [[gc]]

## Context 概述

`context.Context` 是 Go 1.7 引入标准库的核心同步工具，广泛用于控制 goroutine 生命周期、传递请求作用域数据和超时信号。标准库中 `os/exec`、`net`、`database/sql`、`runtime/pprof` 等包均已支持 Context。

Context 的核心特性：
- **并发安全**：可在多个 goroutine 间安全传播
- **树形结构**：通过 parent-child 关系构成树，子节点可继承父节点的属性并响应取消信号
- **接口类型**：底层实现均为指针类型，传播不影响功能和安全性

---

## Context 树形结构

所有 Context 值构成一棵树，树根是 `context.Background()` 返回的全局唯一根节点。根节点不可撤销、不携带数据，仅作为起点。

```mermaid
graph TD
    BG["context.Background()"] --> C1["WithCancel"]
    BG --> V1["WithValue(key1, val1)"]
    C1 --> T1["WithTimeout(5s)"]
    C1 --> V2["WithValue(key2, val2)"]
    T1 --> C2["WithCancel"]
    
    style BG fill:#e0e0e0,stroke:#333
    style C1 fill:#ffcccc,stroke:#333
    style T1 fill:#ffcccc,stroke:#333
    style C2 fill:#ffcccc,stroke:#333
    style V1 fill:#ccffcc,stroke:#333
    style V2 fill:#ccffcc,stroke:#333
```

Context 包提供 4 个派生函数，第一个参数都是 `parent`：

| 函数 | 作用 |
|------|------|
| `WithCancel(parent)` | 创建可手动撤销的子 Context |
| `WithDeadline(parent, d)` | 创建到指定时间自动撤销的子 Context |
| `WithTimeout(parent, d)` | 创建经过指定时长后自动撤销的子 Context |
| `WithValue(parent, k, v)` | 创建携带 key-value 数据的子 Context |

---

## 源码分析（go1.23）

### 接口定义

```go
type Context interface {
    Deadline() (deadline time.Time, ok bool)
    Done() <-chan struct{}
    Err() error
    Value(key any) any
}
```

四个方法的语义：

| 方法 | 说明 |
|------|------|
| `Deadline()` | 返回截止时间和是否已设置 deadline 的 bool 值 |
| `Done()` | 返回一个 channel，Context 关闭后该 channel 被 close，用于 `select` 监听 |
| `Err()` | Context 关闭后返回关闭原因：`"context canceled"` 或 `"context deadline exceeded"` |
| `Value(key)` | 沿 Context 链向上查找 key 对应的 value |

### 典型使用模式

```go
func doWork(ctx context.Context) error {
    // 派生一个 5 秒超时的子 Context
    ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
    defer cancel() // 确保资源释放

    select {
    case <-ctx.Done():
        return ctx.Err() // 超时或被取消
    case result := <-longRunningTask():
        return process(result)
    }
}
```

---

### emptyCtx — 空 Context

`emptyCtx` 是 Context 树的根节点实现，所有方法返回零值，仅作为其他 Context 的父节点。

```go
type emptyCtx struct{}

func (emptyCtx) Deadline() (deadline time.Time, ok bool) { return }
func (emptyCtx) Done() <-chan struct{}                     { return nil }
func (emptyCtx) Err() error                               { return nil }
func (emptyCtx) Value(key any) any                         { return nil }
```

Go 暴露两个函数创建空 Context：

```go
func Background() Context { return backgroundCtx{} }  // 主函数、初始化、顶层请求使用
func TODO() Context       { return todoCtx{} }         // 不确定用哪个 Context 时临时占位
```

---

### cancelCtx — 可撤销 Context

#### 数据结构

```go
type cancelCtx struct {
    Context

    mu       sync.Mutex            // 保护下面的字段
    done     atomic.Value          // chan struct{}，懒创建，首次 cancel 时关闭
    children map[canceler]struct{} // 所有可撤销的子 Context，cancel 时全部撤销
    err      error                 // 首次 cancel 时设置
    cause    error                 // 首次 cancel 时设置
}
```

`children` 记录了所有派生的可撤销子 Context。cancel 时会遍历 children **深度优先**地逐个取消。

#### Done() 实现 — 双重检查锁定

```go
func (c *cancelCtx) Done() <-chan struct{} {
    d := c.done.Load()
    if d != nil {
        return d.(chan struct{}) // fast path: channel 已创建
    }
    c.mu.Lock()
    defer c.mu.Unlock()
    d = c.done.Load()
    if d == nil {
        d = make(chan struct{})  // slow path: 加锁后创建
        c.done.Store(d)
    }
    return d.(chan struct{})
}
```

采用 **fast path + slow path** 模式：先无锁尝试读取，失败后加锁创建，经典的 double-checked locking。

#### Err() 实现

```go
func (c *cancelCtx) Err() error {
    c.mu.Lock()
    err := c.err
    c.mu.Unlock()
    return err
}
```

#### cancel() 实现 — 核心取消逻辑

```mermaid
flowchart TD
    A["cancel(removeFromParent, err, cause)"] --> B{err == nil?}
    B -- Yes --> PANIC["panic"]
    B -- No --> C["c.mu.Lock()"]
    C --> D{c.err != nil?}
    D -- Yes --> E["已取消，直接返回"]
    D -- No --> F["设置 c.err, c.cause"]
    F --> G["关闭 done channel"]
    G --> H["遍历 children，递归 cancel"]
    H --> I["c.children = nil"]
    I --> J["c.mu.Unlock()"]
    J --> K{removeFromParent?}
    K -- Yes --> L["removeChild(parent, c)"]
    K -- No --> M["结束"]
    L --> M
```

源码：

```go
func (c *cancelCtx) cancel(removeFromParent bool, err, cause error) {
    if err == nil {
        panic("context: internal error: missing cancel error")
    }
    if cause == nil {
        cause = err
    }
    c.mu.Lock()
    if c.err != nil {
        c.mu.Unlock()
        return // already canceled
    }
    c.err = err
    c.cause = cause
    d, _ := c.done.Load().(chan struct{})
    if d == nil {
        c.done.Store(closedchan) // 未创建则直接存入已关闭的 channel
    } else {
        close(d)                 // 已创建则关闭
    }
    for child := range c.children {
        child.cancel(false, err, cause) // 递归取消所有子 Context
    }
    c.children = nil
    c.mu.Unlock()

    if removeFromParent {
        removeChild(c.Context, c) // 从父 Context 的 children 中移除自己
    }
}
```

关键点：
- 持有父锁的同时获取子锁（锁顺序固定为 parent → child），不会死锁
- 取消信号**深度优先**传播：先递归取消子节点，再释放自身锁

#### WithCancel() 实现

```go
func WithCancel(parent Context) (ctx Context, cancel CancelFunc) {
    c := withCancel(parent)
    return c, func() { c.cancel(true, Canceled, nil) }
}
```

内部逻辑：
1. 创建 `cancelCtx` 实例
2. 将新 Context 挂载到父节点的 `children`（如果父节点支持 cancel）
3. 如果所有祖先都不支持 cancel，则启动一个 goroutine 监听父节点的 `Done()` channel

---

### timerCtx — 定时撤销 Context

#### 数据结构

```go
type timerCtx struct {
    cancelCtx
    timer    *time.Timer // 定时器，到期自动 cancel
    deadline time.Time   // 截止时间
}
```

在 `cancelCtx` 基础上增加了 timer 和 deadline，衍生出 `WithDeadline()` 和 `WithTimeout()`。

- **deadline**：指定绝对截止时间，如 2024-01-01 00:00:00
- **timeout**：指定相对存活时长，如 30s 后到期

#### Deadline() 实现

```go
func (c *timerCtx) Deadline() (deadline time.Time, ok bool) {
    return c.deadline, true
}
```

#### cancel() 实现

```go
func (c *timerCtx) cancel(removeFromParent bool, err, cause error) {
    c.cancelCtx.cancel(false, err, cause) // 委托给 cancelCtx
    if removeFromParent {
        removeChild(c.cancelCtx.Context, c)
    }
    c.mu.Lock()
    if c.timer != nil {
        c.timer.Stop() // 额外停止定时器
        c.timer = nil
    }
    c.mu.Unlock()
}
```

关闭原因取决于触发方式：
- 手动 cancel → `"context canceled"`
- deadline 到期 → `"context deadline exceeded"`

#### WithDeadline() 实现

```go
func WithDeadlineCause(parent Context, d time.Time, cause error) (Context, CancelFunc) {
    if parent == nil {
        panic("cannot create context from nil parent")
    }
    // 父节点的 deadline 更早，新 deadline 无意义，退化为 WithCancel
    if cur, ok := parent.Deadline(); ok && cur.Before(d) {
        return WithCancel(parent)
    }
    c := &timerCtx{deadline: d}
    c.cancelCtx.propagateCancel(parent, c)
    dur := time.Until(d)
    if dur <= 0 {
        c.cancel(true, DeadlineExceeded, cause) // deadline 已过，立即取消
        return c, func() { c.cancel(false, Canceled, nil) }
    }
    c.mu.Lock()
    defer c.mu.Unlock()
    if c.err == nil {
        c.timer = time.AfterFunc(dur, func() {
            c.cancel(true, DeadlineExceeded, cause)
        })
    }
    return c, func() { c.cancel(true, Canceled, nil) }
}
```

关键逻辑：
1. 如果父节点 deadline 更早，新 deadline 无效，退化为 `WithCancel`
2. 如果 deadline 已过，立即取消
3. 否则启动定时器，到期自动 cancel

#### WithTimeout() 实现

`WithTimeout` 只是 `WithDeadline` 的语法糖：

```go
func WithTimeout(parent Context, timeout time.Duration) (Context, CancelFunc) {
    return WithDeadline(parent, time.Now().Add(timeout))
}
```

---

### valueCtx — 携带数据的 Context

#### 数据结构

```go
type valueCtx struct {
    Context
    key, val any
}
```

每个 `valueCtx` 只存储一个 key-value 对。

#### Value() 实现 — 链式查找

```go
func (c *valueCtx) Value(key any) any {
    if c.key == key {
        return c.val
    }
    return value(c.Context, key) // 递归向父节点查找
}
```

`value()` 函数沿 Context 链向上遍历，直到找到匹配的 key 或到达根节点返回 nil。子 Context 可以查到所有祖先的 value，但祖先看不到后代的 value。

#### WithValue() 实现

```go
func WithValue(parent Context, key, val any) Context {
    if parent == nil {
        panic("cannot create context from nil parent")
    }
    if key == nil {
        panic("nil key")
    }
    if !reflectlite.TypeOf(key).Comparable() {
        panic("key is not comparable")
    }
    return &valueCtx{parent, key, val}
}
```

注意：key 应使用自定义类型（通常是 `struct{}`），避免不同包之间的 key 冲突。

---

## 总结

```mermaid
classDiagram
    class Context {
        <<interface>>
        +Deadline() (time.Time, bool)
        +Done() chan struct{}
        +Err() error
        +Value(key any) any
    }
    class emptyCtx {
    }
    class cancelCtx {
        +mu sync.Mutex
        +done atomic.Value
        +children map
        +err error
        +cancel()
    }
    class timerCtx {
        +timer *time.Timer
        +deadline time.Time
        +cancel()
    }
    class valueCtx {
        +key any
        +val any
        +Value()
    }
    Context <|.. emptyCtx
    Context <|.. cancelCtx
    cancelCtx <|-- timerCtx
    Context <|.. valueCtx
```

| 类型 | 创建函数 | 用途 |
|------|----------|------|
| `emptyCtx` | `Background()` / `TODO()` | 根节点，不携带任何信息 |
| `cancelCtx` | `WithCancel()` | 手动取消，信号向子树传播 |
| `timerCtx` | `WithDeadline()` / `WithTimeout()` | 定时自动取消 |
| `valueCtx` | `WithValue()` | 传递请求作用域数据 |

## 面试要点

### 高频问题

**Q: Context 的接口包含哪四个方法？分别有什么作用？**

> [!question]- 参考答案（点击展开）
>
> `Deadline()` 返回截止时间和是否设置了 deadline 的 bool；`Done()` 返回一个 `<-chan struct{}`，Context 取消后该 channel 被 close，供 `select` 监听；`Err()` 在 Context 关闭后返回原因（`context canceled` 或 `context deadline exceeded`），未关闭时返回 nil；`Value(key)` 沿 Context 链向上查找 key 对应的 value。

**Q: Context 主要用来解决什么问题？典型使用场景有哪些？**

> [!question]- 参考答案（点击展开）
>
> 主要解决 goroutine 的生命周期控制、超时/取消信号传播和请求作用域数据传递。典型场景是 HTTP/RPC 请求链路：上游取消或超时后，通过 Context 把信号扩散到所有下游 goroutine，避免 goroutine 泄漏。标准库 `net`、`database/sql`、`os/exec`、`runtime/pprof` 等都支持传入 Context。

**Q: Background() 和 TODO() 有什么区别？**

> [!question]- 参考答案（点击展开）
>
> 二者返回的具体类型不同（go1.21+ 分别是 `backgroundCtx` 和 `todoCtx`），但都基于同一份 `emptyCtx` 实现——所有方法返回零值，不可取消、不携带数据。区别在语义：`Background()` 用于 main、init 或顶层请求入口，作为 Context 树的根；`TODO()` 用于暂时不确定该传哪个 Context、后续待补全的占位场景，便于代码审查时识别。

**Q: WithCancel 创建的 Context 取消时，取消信号是怎么传播的？**

> [!question]- 参考答案（点击展开）
>
> `cancelCtx` 用 `children map[canceler]struct{}` 记录所有可取消的子 Context。调用 cancel 时，先加锁设置 err/cause、close 掉 done channel，然后遍历 children 递归调用其 cancel，最后把 children 置 nil。信号沿子树**深度优先递归**传播，锁顺序固定为 parent→child，所以不会死锁。

**Q: 为什么每次调用 WithCancel/WithTimeout 后都建议 defer cancel()？不调用会怎样？**

> [!question]- 参考答案（点击展开）
>
> cancel 函数负责释放 Context 关联的资源——把自己从父节点的 children 中移除（`removeChild`），并停止 timerCtx 的 timer。如果不调用，对于挂在可取消父节点下的子 Context，它会一直留在父节点的 children map 里直到父节点取消，造成内存泄漏；timerCtx 的定时器也无法提前回收。`go vet`（lostcancel 检查）会对漏掉 cancel 的代码报警告。

**Q: WithDeadline 和 WithTimeout 是什么关系？**

> [!question]- 参考答案（点击展开）
>
> `WithTimeout(parent, d)` 只是 `WithDeadline(parent, time.Now().Add(d))` 的语法糖。deadline 是绝对时间点，timeout 是相对时长。底层都用 `timerCtx`，通过 `time.AfterFunc` 在到期时触发 cancel，取消原因为 `context deadline exceeded`（哨兵错误 `context.DeadlineExceeded`）。

**Q: 如果父 Context 的 deadline 比子 Context 设置的更早，会发生什么？**

> [!question]- 参考答案（点击展开）
>
> `WithDeadline` 内部会检查：若父节点已有 deadline 且比新设置的更早（`cur.Before(d)`，即父的 `cur` 早于新的 `d`），新 deadline 没有意义，直接退化为 `WithCancel(parent)`，不再单独创建 timer。这样子 Context 仍会随父节点的更早 deadline 一起被取消。

**Q: WithValue 传值有哪些注意事项？Value 查找的性能如何？**

> [!question]- 参考答案（点击展开）
>
> key 必须是可比较类型（不可比较会 panic），且应使用包内自定义类型（如 `type ctxKey struct{}`）而非内置的 `string`/`int`，避免跨包 key 冲突。每个 `valueCtx` 只存一个 kv，`Value()` 是沿链向上**线性递归查找**，链越长越慢，因此 Context 只适合传请求作用域的少量元数据（如 traceID、authToken），不应传函数的可选参数。

### 面试加分点

- `cancelCtx.Done()` 采用 **double-checked locking**：done channel 懒创建，先无锁走 fast path 读 `atomic.Value`，为 nil 再加锁创建 slow path，兼顾性能与并发安全。
- cancel 时对 done channel 有个细节优化：若 channel 尚未被 `Done()` 创建，不会新建再 close，而是直接存入预先构造好的全局 `closedchan`（一个已关闭的 channel），省去一次分配。
- 能说清取消信号是**深度优先递归**传播且锁顺序固定为 parent→child，并解释 `cancel` 的 `removeFromParent` 参数：手动 cancel 时为 true（需从父 children 中摘除自己），递归取消子节点时传 false（父节点稍后会整体把 children 置 nil）。
- `timerCtx` 内嵌 `cancelCtx` 复用其取消逻辑，自己的 cancel 只是委托给 `cancelCtx.cancel` 后额外 `timer.Stop()` 并置 nil，体现组合优于继承的设计。
- 区分 `Err()` 返回的哨兵错误 `context.Canceled` 与 `context.DeadlineExceeded`，并知道 Go 1.20 引入的 `context.Cause(ctx)` / `WithCancelCause` 可携带自定义取消原因，比单纯的 `Err()` 信息更丰富。
- Context 应作为函数的第一个参数显式传递、不要存进 struct 字段；除 `emptyCtx`（值类型）外，`cancelCtx`/`timerCtx`/`valueCtx` 底层都以指针形式传递，跨 goroutine 传播并发安全。
