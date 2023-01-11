# 问题
- 协程是如何从栈上找出引用堆变量的变量的？

猜测：函数运行，如果某个局部变量引用了堆变量，会将此记录到协程相应的数据结构中。 gc 携程只需要扫描工作协程的该变量即可。
如此实现原因：类似 epoll 将 gc 协程扫描工作分摊到各个工作协程。效率较高。其次，如果 gc 协程直接从工作协程栈上找引用堆变量的局部变量，我想不到很好的方法。gc 协程能看到的也就是工作协程的栈顶、栈基这些基本信息。

- mark-sweep 标记-清扫 算法

什么时候算标记完成呢？
清扫阶段，如果工作协程修改了引用关系，需要重新标记吗？

- 既然是三色标记，为什么 go 内部中使用 bit 位来标记？ 这不是只能标记两种状态吗？

0 为白色，未标记
1 且对象入队 为灰色
1 且对象出队，表示对象引用所有对象均标记，为黑色

# moduledata
moduledata 描述了 go 可执行文件的内存布局,包括函数信息,类型元数据信息,每一条 pc 信息等等. moduledata 类型的变量是跟 gc 无关的,所以链接器单独在可执行文件中分配了一个 section `.noptrdata`. 在 `data` 和 `bss` 中的数据都会被 gc 扫描.

我们可以通过 `objdump -h ./a.out` 查看可执行文件都有哪些 section. 可以通过 `objdump -j .noptrdata -t ./a.out` 查看特定 section 都有哪些 symbol. 关于 `` 的使用,可以参考 linux_learn 中相关内容.

为什么要了解 moduledata 呢? 因为 gc 中的栈扫描,会使用到 moduledata 中的信息. gcWorker 在扫描协程栈的时候,需要了解当前函数栈帧的 locals 和 args 中哪些成员是 pointer, 这样才能标记对象,而函数栈帧的信息就保存在 moduledata 中. 另外, 发生 panic 时候的栈回溯也会使用到 moduledata 信息. 具体的逻辑集中在函数 `gentraceback` 中. 我们看看此函数的实现.

# gentraceback
我们主要分析跟栈扫描相关的代码.
```go
// 栈扫描的逻辑发生在 scanstack 中.
func scanstack(gp *g, gcw *gcWork) {
    	// Scan the stack. Accumulate a list of stack objects.
	scanframe := func(frame *stkframe, unused unsafe.Pointer) bool {
		scanframeworker(frame, &state, gcw)
		return true
	}
	gentraceback(^uintptr(0), ^uintptr(0), 0, gp, 0, nil, 0x7fffffff, scanframe, nil, 0)
}

func gentraceback(pc0, sp0, lr0 uintptr, gp *g, skip int, pcbuf *uintptr, max int, callback func(*stkframe, unsafe.Pointer) bool, v unsafe.Pointer, flags uint) int {
	level, _, _ := gotraceback()

	var ctxt *funcval // Context pointer for unstarted goroutines. See issue #25897.

    // 如果 pc0 和 sp0 等于 0  的话,则从 gp 的 g.sched 中获取保存的 pc 和 sp
	if pc0 == ^uintptr(0) && sp0 == ^uintptr(0) { // Signal to fetch saved values from gp.
        pc0 = gp.sched.pc
        sp0 = gp.sched.sp
        ctxt = (*funcval)(gp.sched.ctxt)
	}

	nprint := 0
	var frame stkframe
	frame.pc = pc0
	frame.sp = sp0

	waspanic := false
	cgoCtxt := gp.cgoCtxt

    // 这里就会访问  moduledata 中的相关信息, findfunc() 根据特定的地址,获取 pc 所在函数的信息, 本质是个 _func 类型的数据.
	f := findfunc(frame.pc)
	frame.fn = f

	var cache pcvalueCache

	lastFuncID := funcID_normal
	n := 0
    // max 是栈回溯最大深度
	for n < max {
		// Typically:
		//	pc is the PC of the running function.
		//	sp is the stack pointer at that program counter.
		//	fp is the frame pointer (caller's stack pointer) at that program counter, or nil if unknown.
		//	stk is the stack containing sp.
		//	The caller's program counter is lr, unless lr is zero, in which case it is *(uintptr*)sp.
		f = frame.fn

		// Compute function info flags.
		flag := f.flag

		// Found an actual function.
		// Derive frame pointer and link register.
        // fp 为 caller 的 sp
		if frame.fp == 0 {
			frame.fp = frame.sp + uintptr(funcspdelta(f, frame.pc, &cache))
			if !usesLR {
				// On x86, call instruction pushes return PC before entering new function.
				frame.fp += sys.PtrSize
			}
		}
		var flr funcInfo
        var lrPtr uintptr

        // lr 为 caller 的 pc
        if frame.lr == 0 {
            lrPtr = frame.fp - sys.PtrSize
            frame.lr = uintptr(*(*uintptr)(unsafe.Pointer(lrPtr)))
        }
        
        // 获取 caller 的 funcInfo
        flr = findfunc(frame.lr)
        
        // varp 执行局部变量的起始地址
		frame.varp = frame.fp
		if !usesLR {
			// On x86, call instruction pushes return PC before entering new function.
			frame.varp -= sys.PtrSize
		}

		// Derive size of arguments.
		// Most functions have a fixed-size argument block,
		// so we can use metadata about the function f.
		// Not all, though: there are some variadic functions
		// in package runtime and reflect, and for those we use call-specific
		// metadata recorded by f's caller.
		if callback != nil || printing {
			frame.argp = frame.fp + sys.MinFrameSize
			var ok bool
            // 从 moduledata 中获取函数信息. arglen: 参数长度, argmap: 参数指针位图
			frame.arglen, frame.argmap, ok = getArgInfoFast(f, callback != nil)
			if !ok {
				frame.arglen, frame.argmap = getArgInfo(&frame, f, callback != nil, ctxt)
			}
		}

        // 在 gc 中,这里调用的就是 scanframe 函数
		if callback != nil {
			if !callback((*stkframe)(noescape(unsafe.Pointer(&frame))), v) {
				return n
			}
		}

		n++
    
		// Unwind to next frame.
        // 设置 caller 栈帧信息
		frame.fn = flr
		frame.pc = frame.lr
		frame.lr = 0
		frame.sp = frame.fp
		frame.fp = 0
		frame.argmap = nil
    }

	return n
}
```

gcWorker 在扫描协程栈的时候,被扫描 g 是停止运行的.

# 写屏障
屏障本质上要解决的问题就是 `对象不可达` 的问题.

# STW(Stop The World)
go runtime 中的 STW 是如何实现的呢?

本质上就是设置一个特殊的变量,其他的 p 会在一些特定的节点(比如: schedule())来检查该变量的值, 如果该变量设置的话,则阻塞 p 绑定的 m. 当最后一个 p 对应的 m 阻塞后,便实现了 STW.
在 go runtime 中,这个变量就是 `gcBlackenEnabled`, 详细内容可以函数 `gcStart` 和 `schedule`.

# GC trigger
什么时候会触发 GC 呢? 有三种方式:
- gcTriggerTime: 定时, 有一个专门触发 GC 的协程 `forcegchelper`, 由监控线程定时唤醒. 默认是 2min
- gcTriggerHeap: 分配内存过多
- gcTriggerCycle: 调用 `runtime.GC()` 手动触发 GC

判断某个 trigger 是否触发 GC 的核心逻辑在函数 `gcTrigger.test` 中, 源码如下:
```go
func (t gcTrigger) test() bool {
	switch t.kind {
	case gcTriggerHeap:
		return gcController.heapLive >= gcController.trigger
	case gcTriggerTime:
		if gcController.gcPercent < 0 {
			return false
		}
		// 在标记结束时,设置为当前时间.
		lastgc := int64(atomic.Load64(&memstats.last_gc_nanotime))
		// forcegcperiod 定义为 2min
		return lastgc != 0 && t.now-lastgc > forcegcperiod
	case gcTriggerCycle:
		// t.n > work.cycles, but accounting for wraparound.
		return int32(t.n-work.cycles) > 0
	}
	return true
}
```
我们重点关注下 gcTriggerHeap 类型. 这里涉及若干重要的变量. 

`heapLive` 表示目前分配了多少内存. 实际上在整个 runtime 中,更新 heapLive 的地方很有限, 这也是出于性能的考虑. 我们看看都有哪些地方更新了 heapLive 的值.
- `refill()`

当分配对象的时候, 如果当前 p 的 mcache 对应的 mspan full 的时候, 就需要从全局 mcentral 中获取一个有空闲空间的 span, 并替换 mcache 中的 span, mcache 中原来的 span 会插入到 mcentral 当中.
当从 mcentral 获取一个有空闲空间的 span 的时候, 便更新 heapLive:
```go
	// 实际上只有分配新对象的时候,才更新 heapLive += sizeof(object). 实际上, 为 p.mcache 分配个新的 span, 该 span 剩下的空间在大多数情况下,也会马上用到的,因此为了性能考虑,
	// 我们在这里直接给 heapLive 加上当前 span 空闲的空间.
	// 因为 heapLive 比实际对象占用的内存多一点点,因此会稍微的提前触发 GC
	atomic.Xadd64(&gcController.heapLive, int64(s.npages*pageSize)-int64(usedBytes))
```
实际上, 当每一轮 GC 标记结束的时候, 会设置 heapLive = heapMarked = 存活对象大小的内存. heapLive 的起始值其实就是一轮 GC 标记结束后存活对象的大小,后续分配新的对象的时候, 同步更新, 当 heapLive 大于 trigger 的时候, 触发新一轮 GC.
```go
func gcMarkTermination() {
	gcMark()
}

func gcMark() {
	// Update the marked heap stat.
	gcController.heapMarked = work.bytesMarked

	// Update other GC heap size stats.
	gcController.heapLive = work.bytesMarked
}
```
实际上 trace 分析中, heap 表示的就是 heapLive 的大小. 所以, 在 trace web ui 中, 我们可以清晰的看到 GC 后, heap 会少很多.
```go
func traceHeapAlloc() {
	traceEvent(traceEvHeapAlloc, -1, gcController.heapLive)
}
```

`trigger` 表示当目前对象占用的内存即 heapLive >= 某个阈值 的时候, 触发 GC. 那么该值是如何计算的呢? 实际上, 当每一轮 GC 标记结束的时候,会基于当前标记的大小 `heapMarked` 和 通过 `GOGC` 设置的值计算 trigger 的大小. `heapMarked` 表示本轮 GC 标记的所有对象的大小, `GOGC` 表示当 heapLive 大于 heapMarked 特定百分比的时候, 触发 GC. 
```go
if c.gcPercent >= 0 {
		goal = c.heapMarked + c.heapMarked*uint64(c.gcPercent)/100
	}

	// Set the trigger ratio, capped to reasonable bounds.
	if c.gcPercent >= 0 {
		// 实际上, ratio 会比 c.gcPercent/100 小一些, 这样的目的在于提前触发 GC
		scalingFactor := float64(c.gcPercent) / 100
		// Ensure there's always a little margin so that the
		// mutator assist ratio isn't infinity.
		maxTriggerRatio := 0.95 * scalingFactor
		if triggerRatio > maxTriggerRatio {
			triggerRatio = maxTriggerRatio
		}
		minTriggerRatio := 0.6 * scalingFactor
		if triggerRatio < minTriggerRatio {
			triggerRatio = minTriggerRatio
		}
	}

	c.triggerRatio = triggerRatio

	// Compute the absolute GC trigger from the trigger ratio.
	//
	// We trigger the next GC cycle when the allocated heap has
	// grown by the trigger ratio over the marked heap size.
	trigger := ^uint64(0)
	if c.gcPercent >= 0 {
		// 计算 trigger
		trigger = uint64(float64(c.heapMarked) * (1 + triggerRatio))
		// Don't trigger below the minimum heap size.
		minTrigger := c.heapMinimum

		// 在一些极端情况下, 比如分配了大量内存,但是存活的对象很少, 就可能导致 trigger 值很小, 如果不限制最小值, 就会导致 频繁 GC
		// c.heapMinimum = 4M
		// 这里我们参考下 gcControlerState 中关于 heapMininum 字段的说明,便可明白该字段的含义
		// heapMinimum is the minimum heap size at which to trigger GC.
		// For small heaps, this overrides the usual GOGC*live set rule.
		//
		// When there is a very small live set but a lot of allocation, simply
		// collecting when the heap reaches GOGC*live results in many GC
		// cycles and high total per-GC overhead. This minimum amortizes this
		// per-GC overhead while keeping the heap reasonably small.
		//
		// During initialization this is set to 4MB*GOGC/100. In the case of
		// GOGC==0, this will set heapMinimum to 0, resulting in constant
		// collection even when the heap size is small, which is useful for
		// debugging.
		//heapMinimum uint64
		if trigger < minTrigger {
			trigger = minTrigger
		}
	}

	// Commit to the trigger and goal.
	c.trigger = trigger
```
