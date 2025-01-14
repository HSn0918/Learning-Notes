# 引言

`context.Context`类型（以下简称`Context`类型）是在Go 1.7发布时才被加入到标准库的。而后，标准库中的很多其他代码包都为了支持它而进行了扩展，包括：`os/exec`包、`net`包、`database/sql`包，以及`runtime/pprof`包和`runtime/trace`包，等等。

`Context`类型之所以受到了标准库中众多代码包的积极支持，主要是因为它是一种非常通用的同步工具。它的值不但可以被任意地扩散，而且还可以被用来传递额外的信息和信号。

更具体地说，`Context`类型可以提供一类代表上下文的值。此类值是并发安全的，也就是说它可以被传播给多个goroutine。

由于`Context`类型实际上是一个接口类型，而`context`包中实现该接口的所有私有类型，都是基于某个数据类型的指针类型，所以，如此传播并不会影响该类型值的功能和安全。

`Context`类型的值（以下简称`Context`值）是可以繁衍的，这意味着我们可以通过一个`Context`值产生出任意个子值。这些子值可以携带其父值的属性和数据，也可以响应我们通过其父值传达的信号。

正因为如此，所有的`Context`值共同构成了一颗代表了上下文全貌的树形结构。这棵树的树根（或者称上下文根节点）是一个已经在`context`包中预定义好的`Context`值，它是全局唯一的。通过调用`context.Background`函数，我们就可以获取到它（我在`coordinateWithContext`函数中就是这么做的）。

这里注意一下，这个上下文根节点仅仅是一个最基本的支点，它不提供任何额外的功能。也就是说，它既不可以被撤销（cancel），也不能携带任何数据。

除此之外，`context`包中还包含了四个用于繁衍`Context`值的函数，即：`WithCancel`、`WithDeadline`、`WithTimeout`和`WithValue`。

这些函数的第一个参数的类型都是`context.Context`，而名称都为`parent`。顾名思义，这个位置上的参数对应的都是它们将会产生的`Context`值的父值。

`WithCancel`函数用于产生一个可撤销的`parent`的子值。在`coordinateWithContext`函数中，我通过调用该函数，获得了一个衍生自上下文根节点的`Context`值，和一个用于触发撤销信号的函数。

而`WithDeadline`函数和`WithTimeout`函数则都可以被用来产生一个会定时撤销的`parent`的子值。至于`WithValue`函数，我们可以通过调用它，产生一个会携带额外数据的`parent`的子值。

到这里，我们已经对`context`包中的函数和`Context`类型有了一个基本的认识了。不过这还不够，我们再来扩展一下。

# 源码分析（go1.23）

## 接口定义

在go的context包下，开头就是Context的接口定义

```
type Context interface{
	Deadline() (deadline time.Time, ok bool)
    Done() <-chan struct{}
    Err() error
    Value(key any) any
}
```

基础的context接口只定义了4个方法，下面分别简要说明一下：

- Deadline()

- 该方法返回一个deadline和标识是否已设置deadline的bool值，如果没有设置deadline，则ok == false，此时deadline为一个初始值的time.Time值

- Done()
- 该方法返回一个channel，需要在select-case语句中使用，如”case <-context.Done():”。

当context关闭后，Done()返回一个被关闭的管道，关闭的管道仍然是可读的，据此goroutine可以收到关闭请求；  
当context还未关闭时，Done()返回nil。

- Err()

- 该方法描述context关闭的原因。关闭原因由context实现控制，不需要用户设置。比如Deadline context，关闭原因可能是因为deadline，也可能提前被主动关闭，那么关闭原因就会不同:

- 因deadline关闭：“context deadline exceeded”；
- 因主动关闭： “context canceled”。

当context关闭后，Err()返回context的关闭原因；  
当context还未关闭时，Err()返回nil；

- Value

- 有一种context，它不是用于控制呈树状分布的goroutine，而是用于在树状分布的goroutine间传递信息。
- Value()方法就是用于此种类型的context，该方法根据key值查询map中的value。具体使用后面示例说明。

## 空Context

context包中定义了一个空的context，名为emptyCtx，用于context的根节点，空的context只是简单的实现了Context，本身不包含任何值，仅用于其他context的父节点。

```
type emptyCtx struct{}

func (emptyCtx) Deadline() (deadline time.Time, ok bool) {
	return
}

func (emptyCtx) Done() <-chan struct{} {
	return nil
}

func (emptyCtx) Err() error {
	return nil
}

func (emptyCtx) Value(key any) any {
	return nil
}
```

如何创建这个空context呢，go暴露了两个方法来实现。

```
type backgroundCtx struct{ emptyCtx }

func (backgroundCtx) String() string {
	return "context.Background"
}

type todoCtx struct{ emptyCtx }

func (todoCtx) String() string {
	return "context.TODO"
}

// Background returns a non-nil, empty [Context]. It is never canceled, has no
// values, and has no deadline. It is typically used by the main function,
// initialization, and tests, and as the top-level Context for incoming
// requests.
func Background() Context {
	return backgroundCtx{}
}

// TODO returns a non-nil, empty [Context]. Code should use context.TODO when
// it's unclear which Context to use or it is not yet available (because the
// surrounding function has not yet been extended to accept a Context
// parameter).
func TODO() Context {
	return todoCtx{}
}
```

在go官方注释中，如果你要创建一个跟context请使用Background方法；当你不知道该传入什么context时候请使用TODO方法。

context包提供了4个方法创建不同类型的context，使用这四个方法时如果没有父context，都需要传入backgroud，即backgroud作为其父节点：

- WithCancel()
- WithDeadline()
- WithTimeout()
- WithValue()

context包中实现Context接口的struct，除了emptyCtx外，还有cancelCtx、timerCtx和valueCtx三种，正是基于这三种context实例，实现了上述4种类型的context。

## cancelCtx

源码中定义了cancelCtx

```
type cancelCtx struct {
	Context

	mu       sync.Mutex            // protects following fields
	done     atomic.Value          // of chan struct{}, created lazily, closed by first cancel call
	children map[canceler]struct{} // set to nil by the first cancel call
	err      error                 // set to non-nil by the first cancel call
	cause    error                 // set to non-nil by the first cancel call
}
```

children中记录了由此context派生的所有child，此context被cancel时会把其中的所有child都cancel掉。

cancelCtx与deadline和value无关，所以只需要实现Done()和Err()外露接口即可。

根据cancelCtx的数据结构，你能不能从中知道`Context`值在传达撤销信号的时候是广度优先的，还是深度优先的？其优势和劣势都是什么？

### Done()接口实现

按照Context定义，Done()接口只需要返回一个channel即可，对于cancelCtx来说只需要返回成员变量done即可。

这里直接看下源码，非常简单：

```
func (c *cancelCtx) Done() <-chan struct{} {
	d := c.done.Load()
	if d != nil {
		return d.(chan struct{})
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	d = c.done.Load()
	if d == nil {
		d = make(chan struct{})
		c.done.Store(d)
	}
	return d.(chan struct{})
}
```

这里使用快慢路径加载。

```
d := c.done.Load()
if d != nil {
    return d.(chan struct{})
}
```

这段代码尝试从 c.done 中加载已经存储的通道。如果通道已经存在（即 d != nil），直接返回它。这一步的作用是快速路径（fast path）检查，以避免不必要的锁操作，从而提高性能。

```
	c.mu.Lock()
	defer c.mu.Unlock()
	d = c.done.Load()
	if d == nil {
		d = make(chan struct{})
		c.done.Store(d)
	}
	return d.(chan struct{})
```

如果 c.done 中的通道为 nil，则需要创建一个新的通道。在创建通道之前，先锁定 c.mu，以确保并发安全。defer c.mu.Unlock() 确保方法退出时释放锁。

锁定后再次检查 c.done 是否已经被其他 goroutine 设置了通道（这是经典的 “双重检查锁定” 模式），如果仍然为 nil，则创建一个新的通道，并存储在 c.done 中。

### Err()接口实现

按照Context定义，Err()只需要返回一个error告知context被关闭的原因。对于cancelCtx来说只需要返回成员变量err即可。

还是直接看下源码：

```
func (c *cancelCtx) Err() error {
	c.mu.Lock()
	err := c.err
	c.mu.Unlock()
	return err
}
```

### cancel()接口实现

cancel()内部方法是理解cancelCtx的最关键的方法，其作用是关闭自己和其后代，其后代存储在cancelCtx.children的map中，其中key值即后代对象，value值并没有意义，这里使用map只是为了方便查询而已。

```
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
		c.done.Store(closedchan)
	} else {
		close(d)
	}
	for child := range c.children {
		// NOTE: acquiring the child's lock while holding parent's lock.
		child.cancel(false, err, cause)
	}
	c.children = nil
	c.mu.Unlock()

	if removeFromParent {
		removeChild(c.Context, c)
	}
}
```

#### **主要逻辑和步骤**

1. **参数检查和初始化**：

```
if err == nil {
    panic("context: internal error: missing cancel error")
}
if cause == nil {
    cause = err
}
```

- 首先检查传入的 err 是否为 nil。如果 err 为空，直接触发 panic，因为取消操作必须伴随一个错误。
- 如果 cause 参数未提供（即 nil），则将其设置为 err。

2. **锁定当前 cancelCtx**：

```
c.mu.Lock()
if c.err != nil {
    c.mu.Unlock()
    return // already canceled
}
```

- 锁定 c.mu，确保对 cancelCtx 的并发修改是安全的。
- 检查 c.err 是否已经设置。如果已经设置，说明 context 已经被取消过，直接返回以避免重复取消。

3. **设置取消状态**：

```
c.err = err
c.cause = cause
d, _ := c.done.Load().(chan struct{})
if d == nil {
    c.done.Store(closedchan)
} else {
    close(d)
}
```

- 将 c.err 设置为取消时传入的 err，并设置 c.cause。
- 从 c.done 中加载通道，如果通道不存在（即 nil），将其设置为一个预定义的关闭通道 closedchan（这个通道已被关闭，用于表示取消状态）。如果通道存在，则关闭它以通知监听者 context 已经取消。

4. **传播取消信号到子 context**：

```
for child := range c.children {
    // NOTE: acquiring the child's lock while holding parent's lock.
    child.cancel(false, err, cause)
}
c.children = nil
```

- 通过遍历 c.children 中的所有子 context，递归地调用子 context 的 cancel() 方法，传递取消信号。
- 注意：这里是在持有父 context 的锁时递归调用子 context 的 cancel()。这种设计需要确保不会引发死锁（deadlock），因为子 context 在取消时也会试图获取自身的锁。

5. **清理和解锁**

```
c.mu.Unlock()
```

- 取消信号传播完毕后，将 c.children 设为空，并释放锁。

6. **从父 context 中移除自己**：

```
if removeFromParent {
    removeChild(c.Context, c)
}
```

- 如果 removeFromParent 参数为 true，调用 removeChild() 函数将自己从父 context 中移除。这一步通常用于在 context 被取消后，防止父 context 再次尝试操作它。

实际上，WithCancel()返回的第二个用于cancel context的方法正是此cancel()。

### WithCancel()方法实现

WithCancel()方法作了三件事：

- 初始化一个cancelCtx实例
- 将cancelCtx实例添加到其父节点的children中(如果父节点也可以被cancel的话)
- 返回cancelCtx实例和cancel()方法

```
type CancelFunc func()

// WithCancel returns a copy of parent with a new Done channel. The returned
// context's Done channel is closed when the returned cancel function is called
// or when the parent context's Done channel is closed, whichever happens first.
//
// Canceling this context releases resources associated with it, so code should
// call cancel as soon as the operations running in this [Context] complete.
func WithCancel(parent Context) (ctx Context, cancel CancelFunc) {
	c := withCancel(parent)
	return c, func() { c.cancel(true, Canceled, nil) }
}
```

这里将自身添加到父节点的过程有必要简单说明一下：

1. 如果父节点也支持cancel，也就是说其父节点肯定有children成员，那么把新context添加到children里即可；
2. 如果父节点不支持cancel，就继续向上查询，直到找到一个支持cancel的节点，把新context添加到children里；
3. 如果所有的父节点均不支持cancel，则启动一个协程等待父节点结束，然后再把当前context结束。

## timerCtx

源码中定义

```
// A timerCtx carries a timer and a deadline. It embeds a cancelCtx to
// implement Done and Err. It implements cancel by stopping its timer then
// delegating to cancelCtx.cancel.
type timerCtx struct {
	cancelCtx
	timer *time.Timer // Under cancelCtx.mu.

	deadline time.Time
}
```

timerCtx在cancelCtx基础上增加了deadline用于标示自动cancel的最终时间，而timer就是一个触发自动cancel的定时器。

由此，衍生出WithDeadline()和WithTimeout()。实现上这两种类型实现原理一样，只不过使用语境不一样：

- deadline: 指定最后期限，比如context将2018.10.20 00:00:00之时自动结束
- timeout: 指定最长存活时间，比如context将在30s后结束。

对于接口来说，timerCtx在cancelCtx基础上还需要实现Deadline()和cancel()方法，其中cancel()方法是重写的。

### Deadline()接口实现

Deadline()方法仅仅是返回timerCtx.deadline而矣。而timerCtx.deadline是WithDeadline()或WithTimeout()方法设置的。

```
func (c *timerCtx) Deadline() (deadline time.Time, ok bool) {
	return c.deadline, true
}
```

### cancel()接口实现

```
func (c *timerCtx) cancel(removeFromParent bool, err, cause error) {
    c.cancelCtx.cancel(false, err, cause)
    if removeFromParent {
        // Remove this timerCtx from its parent cancelCtx's children.
        removeChild(c.cancelCtx.Context, c)
    }
    c.mu.Lock()
    if c.timer != nil {
        c.timer.Stop()
        c.timer = nil
    }
    c.mu.Unlock()
}
```

cancel()方法基本继承cancelCtx，只需要额外把timer关闭。

timerCtx被关闭后，timerCtx.cancelCtx.err将会存储关闭原因：

- 如果deadline到来之前手动关闭，则关闭原因与cancelCtx显示一致；
- 如果deadline到来时自动关闭，则原因为：”context deadline exceeded”

### WithDeadline()方法实现

```
func WithDeadline(parent Context, d time.Time) (Context, CancelFunc) {
	return WithDeadlineCause(parent, d, nil)
}
// WithDeadlineCause behaves like [WithDeadline] but also sets the cause of the
// returned Context when the deadline is exceeded. The returned [CancelFunc] does
// not set the cause.
func WithDeadlineCause(parent Context, d time.Time, cause error) (Context, CancelFunc) {
	if parent == nil {
		panic("cannot create context from nil parent")
	}
	if cur, ok := parent.Deadline(); ok && cur.Before(d) {
		// The current deadline is already sooner than the new one.
		return WithCancel(parent)
	}
	c := &timerCtx{
		deadline: d,
	}
	c.cancelCtx.propagateCancel(parent, c)
	dur := time.Until(d)
	if dur <= 0 {
		c.cancel(true, DeadlineExceeded, cause) // deadline has already passed
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

```
func WithDeadline(parent Context, d time.Time) (Context, CancelFunc) {
    return WithDeadlineCause(parent, d, nil)
}
```

- `WithDeadline` 是 `WithDeadlineCause` 的包装器。它调用 `WithDeadlineCause`，并传递 `nil` 作为 `cause` 参数。因此，当使用 `WithDeadline` 时，超过截止时间时不会记录具体的取消原因。

```
func WithDeadlineCause(parent Context, d time.Time, cause error) (Context, CancelFunc) {
    if parent == nil {
        panic("cannot create context from nil parent")
    }
```

- 首先检查 `parent` 是否为 `nil`。如果 `parent` 为 `nil`，则直接引发 `panic`，因为所有 `Context` 都必须有一个非 `nil` 的父上下文（除了 `context.Background()` 和 `context.TODO()`）。

```
    if cur, ok := parent.Deadline(); ok && cur.Before(d) {
        // The current deadline is already sooner than the new one.
        return WithCancel(parent)
    }
```

- 检查 `parent` 是否已经有一个截止时间 (`cur`)。如果 `parent` 的截止时间比新设置的 `deadline` 早，则直接使用 `WithCancel` 创建一个可取消的 `Context`，因为在这种情况下，新设置的截止时间无效（`parent` 的截止时间会更早发生取消）。

```
    c := &timerCtx{
        deadline: d,
    }
    c.cancelCtx.propagateCancel(parent, c)
```

- 创建一个 `timerCtx` 结构体，这是一个包含截止时间的 `Context` 实现，并调用 `propagateCancel` 来将这个新创建的 `Context` 附加到 `parent` 上，以便在 `parent` 被取消时，`c` 也能收到取消信号。

```
    dur := time.Until(d)
    if dur <= 0 {
        c.cancel(true, DeadlineExceeded, cause) // deadline has already passed
        return c, func() { c.cancel(false, Canceled, nil) }
    }
```

- 计算从现在到指定截止时间的持续时间 (`dur`)。如果 `dur` 小于等于 0，说明截止时间已经过去，因此立即取消该 `Context`，并设置取消原因为 `DeadlineExceeded`。随后返回这个已取消的 `Context` 和一个无效的 `CancelFunc`（可以手动调用，但不会有实际效果）。

```
    c.mu.Lock()
    defer c.mu.Unlock()
    if c.err == nil {
        c.timer = time.AfterFunc(dur, func() {
            c.cancel(true, DeadlineExceeded, cause)
        })
    }
```

- 如果截止时间还未到达，则锁定 `c.mu`，确保对 `c.timer` 的操作是并发安全的。
- 如果 `c.err` 为 `nil`，即 `Context` 尚未被取消，则设置一个定时器 `c.timer`。当定时器触发时（即达到截止时间时），调用 `c.cancel`，取消 `Context` 并记录 `cause`（如果有的话）。

```
    return c, func() { c.cancel(true, Canceled, nil) }
```

- 最后返回创建的 `Context` 和 `CancelFunc`。`CancelFunc` 允许调用者手动取消 `Context`，设置的取消原因将是 `Canceled`，但不会附加任何 `cause`。

### WithTimeout()方法实现

WithTimeout()实际调用了WithDeadline，二者实现原理一致。

```
// WithTimeout returns WithDeadline(parent, time.Now().Add(timeout)).
//
// Canceling this context releases resources associated with it, so code should
// call cancel as soon as the operations running in this [Context] complete:
//
//	func slowOperationWithTimeout(ctx context.Context) (Result, error) {
//		ctx, cancel := context.WithTimeout(ctx, 100*time.Millisecond)
//		defer cancel()  // releases resources if slowOperation completes before timeout elapses
//		return slowOperation(ctx)
//	}
func WithTimeout(parent Context, timeout time.Duration) (Context, CancelFunc) {
	return WithDeadline(parent, time.Now().Add(timeout))
}

// WithTimeoutCause behaves like [WithTimeout] but also sets the cause of the
// returned Context when the timeout expires. The returned [CancelFunc] does
// not set the cause.
func WithTimeoutCause(parent Context, timeout time.Duration, cause error) (Context, CancelFunc) {
	return WithDeadlineCause(parent, time.Now().Add(timeout), cause)
}
```

## valueCtx

源码中定义

```
// A valueCtx carries a key-value pair. It implements Value for that key and
// delegates all other calls to the embedded Context.
type valueCtx struct {
	Context
	key, val any
}
```

valueCtx只是在Context基础上增加了一个key-value对，用于在各级协程间传递一些数据。

由于valueCtx既不需要cancel，也不需要deadline，那么只需要实现Value()接口即可。

### Value（）接口实现

```
func (c *valueCtx) Value(key any) any {
	if c.key == key {
		return c.val
	}
	return value(c.Context, key)
}

func value(c Context, key any) any {
	for {
		switch ctx := c.(type) {
		case *valueCtx:
			if key == ctx.key {
				return ctx.val
			}
			c = ctx.Context
		case *cancelCtx:
			if key == &cancelCtxKey {
				return c
			}
			c = ctx.Context
		case withoutCancelCtx:
			if key == &cancelCtxKey {
				// This implements Cause(ctx) == nil
				// when ctx is created using WithoutCancel.
				return nil
			}
			c = ctx.c
		case *timerCtx:
			if key == &cancelCtxKey {
				return &ctx.cancelCtx
			}
			c = ctx.Context
		case backgroundCtx, todoCtx:
			return nil
		default:
			return c.Value(key)
		}
	}
}
```

这里有个细节需要关注一下，即当前context查找不到key时，会向父节点查找，如果查询不到则最终返回interface{}。也就是说，可以通过子context查询到父的value值。

### WithValue（）方法实现

WithValue()实现也是非常的简单, 代码如下：

```
// WithValue returns a copy of parent in which the value associated with key is
// val.
//
// Use context Values only for request-scoped data that transits processes and
// APIs, not for passing optional parameters to functions.
//
// The provided key must be comparable and should not be of type
// string or any other built-in type to avoid collisions between
// packages using context. Users of WithValue should define their own
// types for keys. To avoid allocating when assigning to an
// interface{}, context keys often have concrete type
// struct{}. Alternatively, exported context key variables' static
// type should be a pointer or interface.
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