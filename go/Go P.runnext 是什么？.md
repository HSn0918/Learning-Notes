下面示例的输出是什么？

```
// go 1.14
package main

import (
	"fmt"
	"os"
	"runtime"
	"runtime/trace"
	"sync"
)

func main() {
	runtime.GOMAXPROCS(1)
	wg := &sync.WaitGroup{}
	wg.Add(3)
	go func() {
		fmt.Println("A")
		wg.Done()
	}()

	go func() {
		fmt.Println("B")
		wg.Done()
	}()

	go func() {
		fmt.Println("C")
		wg.Done()
	}()

	wg.Wait()
}
```

先说结论，如下，且是稳定的

```
// OUTPUT:
// C
// A
// B
```

涉及如下知识点：

1. `runtime.GOMAXPROCS()`

GOMAXPROCS sets the maximum number of CPUs that can be executing simultaneously and returns the previous setting.

1. 设置同时可执行的`cpu`数量，当设置为`1`时，可以认为是单线程
2. `sync.WaitGroup{}`  
    一种等待同步机制，`wg.Wait()`会一直等待`wg`的值为`0`才会继续执行。`wg.Add()`增加，`wg.Done()`减一。
3. `go`关键字都做了什么？  
    这里需要对Go的GMP调度模型有一定了解，[推荐 [典藏版] Golang 调度器 GMP 原理与调度全分析](https://learnku.com/articles/41728)  
    go关键字在代码生成过程中会被转换为`runtime.newproc`

```
//`$GOROOT/src/cmd/compile/internal/gc/ssa.go`

case k == callGo:
	call = s.newValue1A(ssa.OpStaticCall, types.TypeMem, newproc, s.mem())
```

```
//`$GOROOT/src/runtime/proc.go`

func newproc(siz int32, fn *funcval) {
	argp := add(unsafe.Pointer(&fn), sys.PtrSize)
	gp := getg()
	pc := getcallerpc()
	systemstack(func() {
		newproc1(fn, argp, siz, gp, pc)
	})
}

// Create a new g running fn with narg bytes of arguments starting
// at argp. callerpc is the address of the go statement that created
// this. The new g is put on the queue of g's waiting to run.
func newproc1(fn *funcval, argp unsafe.Pointer, narg int32, callergp *g, callerpc uintptr) {
	// 省略部分代码
  
	// 获取当前 goroutine
	_g_ := getg()
	// 获取当前 goroutine 所在的 P
  _p_ := _g_.m.p.ptr()
	// 生成新创建的 goroutine 数据结构
	newg := gfget(_p_)
  
	// 将新创建的 goroutine 状态设置为`_Grunnable`
	casgstatus(newg, _Gdead, _Grunnable)
	// 生成新创建的 goroutine id
	newg.goid = int64(_p_.goidcache)
	// 将新创建的  goroutine 放入 “当前 goroutine 所在的 P” 的本地队列中，注意第三个参数为`true`
 	runqput(_p_, newg, true)

	if atomic.Load(&sched.npidle) != 0 && atomic.Load(&sched.nmspinning) == 0 && mainStarted {
		wakep()
	}
	releasem(_g_.m)
}
```

4. `P.runnext`是什么？  
    每个P上都有一个`runnext`字段，类型是`guintptr`，语义为下一个优先执行的`goroutine`  
    但是只能存储一个，如果有多个，被抢占掉的`goroutine`会被放到可执行队列的队尾，等到正常调度

```
//`$GOROOT/src/runtime/proc.go`

// runqput tries to put g on the local runnable queue.
// If next is false, runqput adds g to the tail of the runnable queue.
// If next is true, runqput puts g in the _p_.runnext slot.
// If the run queue is full, runnext puts g on the global queue.
// Executed only by the owner P.
func runqput(_p_ *p, gp *g, next bool) {
	if next {
	retryNext:
		oldnext := _p_.runnext
		// 将 _p_.runnext 的旧值和当前 goroutine 进行交换
		if !_p_.runnext.cas(oldnext, guintptr(unsafe.Pointer(gp))) {
			goto retryNext
		}
		// 如果 _p_.runnext 的旧值为空，则直接返回
		if oldnext == 0 {
			return
		}
		// Kick the old runnext out to the regular run queue.
    // 如果 _p_.runnext 的旧值不为空，获取其对应的 goroutine 指针，gp 为_p_.runnext 的旧值
		gp = oldnext.ptr()
	}
// 如果 next 为 false ，则 gp 仍为新创建的 goroutine
// 如果 next 为 true ，则 gp 为_p_.runnext 的旧值
// 将 gp 放置到 P 本地队列的队尾，如果 P 的本地队列已满，则放置到全局队列上，等待调度器进行调度
retry:
	h := atomic.LoadAcq(&_p_.runqhead) // load-acquire, synchronize with consumers
	t := _p_.runqtail
	if t-h < uint32(len(_p_.runq)) {
		_p_.runq[t%uint32(len(_p_.runq))].set(gp)
		atomic.StoreRel(&_p_.runqtail, t+1) // store-release, makes the item available for consumption
		return
	}
	if runqputslow(_p_, gp, h, t) {
		return
	}
	// the queue is not full, now the put above must succeed
	goto retry
}
```

调度相关的任务，分为 每个 P 上的本地队列（减少锁冲突的一种优化）、全局队列（本地队列已满或者P需要被释放时，会将任务放置到全局队列中）、还有本文所涉及的 `P.runnext`

5. 调度器视角下的任务优先级，调度器的函数为`runtime.schedule()`，该函数的目的是找到一个可执行的`goroutine`去运行，在该函数中，获取`goroutine`的位置越靠前，则可以认为其优先级越高。

```
//`$GOROOT/src/runtime/proc.go`

// One round of scheduler: find a runnable goroutine and execute it.
// Never returns.
func schedule() {
	_g_ := getg()

  // runtime.LockOSThread()相关，将G和M进行双向绑定
	if _g_.m.lockedg != 0 {
		stoplockedm()
		execute(_g_.m.lockedg.ptr(), false) // Never returns.
	}

top:
	pp := _g_.m.p.ptr()
	pp.preempt = false
	// 如果处于 Stop The World 等待过程中，则优先进行 STW，等待Start The World 后再进行调度
	if sched.gcwaiting != 0 {
		gcstopm()
		goto top
	}
  
	// 在go1.14之后，timer的运行检查放置到 schedule 中
	checkTimers(pp, 0)

	var gp *g
	var inheritTime bool
 
	// Normal goroutines will check for need to wakeP in ready,
	// but GCworkers and tracereaders will not, so the check must
	// be done here instead.
	tryWakeP := false
	// trace相关goroutine
	if trace.enabled || trace.shutdown {
		gp = traceReader()
		if gp != nil {
			casgstatus(gp, _Gwaiting, _Grunnable)
			traceGoUnpark(gp, 0)
			tryWakeP = true
		}
	}
	// gc相关goroutine
	if gp == nil && gcBlackenEnabled != 0 {
		gp = gcController.findRunnableGCWorker(_g_.m.p.ptr())
		tryWakeP = tryWakeP || gp != nil
	}
	// 下面开始正常 goroutine 的查找（越靠前的优先级越高）
	if gp == nil {
		// Check the global runnable queue once in a while to ensure fairness.
		// Otherwise two goroutines can completely occupy the local runqueue
		// by constantly respawning each other.
		// 为了防止全局队列“饿死”，每个P 每处理 61 个goroutine，会从全局获取一个goroutine进行处理
		if _g_.m.p.ptr().schedtick%61 == 0 && sched.runqsize > 0 {
			lock(&sched.lock)
			gp = globrunqget(_g_.m.p.ptr(), 1)
			unlock(&sched.lock)
		}
	}
	// 从 P.runnext 和 P的本地队列中进行寻找
	if gp == nil {
		gp, inheritTime = runqget(_g_.m.p.ptr())
		// We can see gp != nil here even if the M is spinning,
		// if checkTimers added a local goroutine via goready.
	}
	// 这个函数比较复杂，再继续检查timer、从P的本地队列、全局队列、netpoll、偷取其他P的本地队列、等进行获取
	if gp == nil {
		gp, inheritTime = findrunnable() // blocks until work is available
	}

	// This thread is going to run a goroutine and is not spinning anymore,
	// so if it was marked as spinning we need to reset it now and potentially
	// start a new spinning M.
	if _g_.m.spinning {
		resetspinning()
	}

	// If about to schedule a not-normal goroutine (a GCworker or tracereader),
	// wake a P if there is one.
	if tryWakeP {
		if atomic.Load(&sched.npidle) != 0 && atomic.Load(&sched.nmspinning) == 0 {
			wakep()
		}
	}
	// runtime.LockOSThread()相关
	if gp.lockedm != 0 {
		// Hands off own p to the locked m,
		// then blocks waiting for a new p.
		startlockedm(gp)
		goto top
	}
	// 运行所找到的 goroutine
	execute(gp, inheritTime)
}
```

```
// Get g from local runnable queue.
// If inheritTime is true, gp should inherit the remaining time in the
// current time slice. Otherwise, it should start a new time slice.
// Executed only by the owner P.
func runqget(_p_ *p) (gp *g, inheritTime bool) {
	// If there's a runnext, it's the next G to run.
	for {
		// 优先从_p_.runnext进行查找，这也就是为什么 _p_.runnext 比其他 goroutine 优先级高的原因
		next := _p_.runnext
		if next == 0 {
			break
		}
		if _p_.runnext.cas(next, 0) {
			return next.ptr(), true
		}
	}
	// 从本地队列进行查找
	for {
		h := atomic.LoadAcq(&_p_.runqhead) // load-acquire, synchronize with other consumers
		t := _p_.runqtail
		if t == h {
			return nil, false
		}
		gp := _p_.runq[h%uint32(len(_p_.runq))].ptr()
		if atomic.CasRel(&_p_.runqhead, h, h+1) { // cas-release, commits consume
			return gp, false
		}
	}
}
```

5. 综上所述，`Goroutine` 所涉及的优先级为：`P.runnext` > `P.localrunq` > `globalrunq`  
    ![](https://cdn.nlark.com/yuque/0/2020/png/1230569/1601604813199-a0828ddf-25fc-4253-8e49-9333c0982a1c.png)  
    所以上述例子中，goroutine的提交顺序为：

```
// go 1.14
package main

import (
	"fmt"
	"os"
	"runtime"
	"runtime/trace"
	"sync"
)

func main() {
	runtime.GOMAXPROCS(1)
	wg := &sync.WaitGroup{}
	wg.Add(3)
	// 提交G1
	go func() {
		fmt.Println("A")
		wg.Done()
	}()
	// 提交G2
	go func() {
		fmt.Println("B")
		wg.Done()
	}()
	// 提交G3
	go func() {
		fmt.Println("C")
		wg.Done()
	}()

	wg.Wait()
}
```

5. ![](https://cdn.nlark.com/yuque/0/2020/png/1230569/1601604813254-f6f8dc9b-861a-406c-a521-6508892e6e74.png)  
    所以`goroutine`最终的执行顺序为`G3 -> G1 ->G2`  
    在`Go1.13`中，可能如下的`demo`会推翻我们上面的结论

```
// go1.13
package main

import (
	"fmt"
	"runtime"
	"time"
)

func main() {
	runtime.GOMAXPROCS(1)
	go func() {
		fmt.Println("A")
	}()

	go func() {
		fmt.Println("B")
	}()

	go func() {
		fmt.Println("C")
	}()
	time.Sleep(time.Second)
}
```

5. 输出如下，且是稳定的

```
// OUTPUT:
// A
// B
// C
```

5. 别急，事出反常必有妖~  
    我们来通过`trace`看一下发生了什么！  
    ![](https://cdn.nlark.com/yuque/0/2020/png/1230569/1601604813362-43a216cb-989e-4a3e-ab9c-0bc5f7812cf5.png)  
    由`trace`可以看出第一个运行的`goroutine`为`runtime.timerproc()`

```
//`$GOROOT/src/runtime/time.go`

// timeSleep puts the current goroutine to sleep for at least ns nanoseconds.
//go:linkname timeSleep time.Sleep
func timeSleep(ns int64) {
	if ns <= 0 {
		return
	}

	gp := getg()
	t := gp.timer
	if t == nil {
		t = new(timer)
		gp.timer = t
	}
	*t = timer{}
	t.when = nanotime() + ns
	t.f = goroutineReady
	t.arg = gp
	tb := t.assignBucket()
	lock(&tb.lock)
	if !tb.addtimerLocked(t) {
		unlock(&tb.lock)
		badTimer()
	}
	goparkunlock(&tb.lock, waitReasonSleep, traceEvGoSleep, 2)
}

// Add a timer to the heap and start or kick timerproc if the new timer is
// earlier than any of the others.
// Timers are locked.
// Returns whether all is well: false if the data structure is corrupt
// due to user-level races.
func (tb *timersBucket) addtimerLocked(t *timer) bool {
	// when must never be negative; otherwise timerproc will overflow
	// during its delta calculation and never expire other runtime timers.
	if t.when < 0 {
		t.when = 1<<63 - 1
	}
	t.i = len(tb.t)
	tb.t = append(tb.t, t)
	if !siftupTimer(tb.t, t.i) {
		return false
	}
	if t.i == 0 {
		// siftup moved to top: new earliest deadline.
		if tb.sleeping && tb.sleepUntil > t.when {
			tb.sleeping = false
			notewakeup(&tb.waitnote)
		}
		if tb.rescheduling {
			tb.rescheduling = false
			goready(tb.gp, 0)
		}
		if !tb.created {
			tb.created = true
			go timerproc(tb)	// 注意，这里隐含启动了一个goroutine
		}
	}
	return true
}
```

5. 见上方注释，在`go1.13`中，对于每个`timer bucket` 都会启动一个后台`goroutine`，来监测自己`bucket`上的`timer`是否需要执行。  
    所以对于上图，在`go1.13`中会新增一个`goroutine`，变为  
    ![](https://cdn.nlark.com/yuque/0/2020/png/1230569/1601604813396-772cbe8e-4ade-456a-b4c4-2eec16c9edd9.png)  
    最后一个问题，都有那些情况下，goroutine由这个“插队”（提交到`P.runnext`）的机会  
    ![](https://cdn.nlark.com/yuque/0/2020/png/1230569/1601604813372-92273cf2-26c4-4612-9a4e-407de0b62b13.png)