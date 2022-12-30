# 疑问
- m 是如何创建的?什么时候会创建一个 m?
- m 休眠是怎么实现的? futex? condition?
- m 的数量有没有限制呢
- gmp 数据结构是什么样的呢?
- g 是如何运行和保存的
- 如何获取当前的 g
- 抢占式调度是如何实现的?
- 怎么理解 p 呢?
- 系统调用的时候,会发生什么?
# 详述
- m

m 是 `Machine` 的简称,表达的是系统线程. 整个 go 程序基于 `OS Thread` + `GMP` 实现各个执行流的调度. 那么在 go 内部,什么时候会创建 thread 呢? 是如何创建 thread 的呢? 以及创建的 thread 执行什么样的函数呢? thread 的数量有没有什么限制呢?

  - 什么时候会创建 thread 呢?

    当创建了新的 goroutine, 当前存在空闲的 p 且 m 数量没有达到上限的时候,就会创建新的 m.尝试创建 thread 是发生在函数 `wakep()`中的. 我们看看该函数的详细实现:
    ```go
    func wakep() {
        // 如果没有空闲的 p, 则直接返回
        if atomic.Load(&sched.npidle) == 0 {
            return
        }

        // 如果存在自旋的 m, 则直接返回
        if atomic.Load(&sched.nmspinning) != 0 {
            return
        }

        // 否则调用 startm() 开启一个新的系统线程
        startm(nil, true)
    }

    // 调度一个新的线程跟 p 绑定
    func startm(_p_ *p, spinning bool) {
        // 如果 _p_ == nil, 则从空闲列表获取 p
        if _p_ == nil {
            _p_ = pidleget()
        }

        // 首先尝试从 m 空闲列表获取 m
        nmp := mget()

        if nmp == nil {
            // 这里会检查 m 的数量,如果超过 sched.maxmcount,则 panic
            id := mReserveID()
            // 调用 newm 生成一个新的 m
            newm(fn, _p_, id)
            return
        }
        ...
    }

    func newm(fn func(), _p_ *p, id int64) {
        // 创建 m struct
        mp := allocm(_p_, fn, id)
        mp.nextp.set(_p_)

        // 调用 newm1 创建线程
        newm1(mp)
    }

    // 创建 m, 初始化 g0(主要初始化栈)
    func allocm(_p_ *p, fn func(), id int64) *m {
        mp := new(m)
        mp.startfn = fn
        mcommoninit(mp, id)

        // malg 分配一个协程
        mp.g0 = malg(8192 * sys.StackGuardMultiplier)
        mp.g0.m = mp

        return mp
    }

    // malg 分配一个协程并初始化协程栈
    func malg(stacksize int32) *g {
        newg := new(g)
        if stacksize > 0 {
            systemstack(func(){
                // 分配栈,新线程的栈实际上就是 go 的栈,便是在这里分配
                newg.stack = stackalloc(stacksize)
            })
            newg.stackguard0 = newg.stack.lo + _StackGuard
            newg.stackguard1 = ^uintptr(0)
        }
    }

    func newm1(mp *m) {
        newosproc(mp)
    }

    func newosproc(mp *m) {
        // stk 就是 g0 对应的栈
        stk := unsafe.Pointer(mp.go.stack.hi)
        // 调用 clone() 系统调用创建线程,线程执行函数为 mstart()
        // c 中的 pthread_create 底层也是通过 clone() 来创建系统线程的
        // mp.g0 就是 tls 数据
        clone(cloneFlags, stk, unsafe.Pointer(mp), unsafe.Pointer(mp.g0), unsafe.Pointer(funcPC(mstart)))
    }

    // mstart -> mstart0() -> mstart1()
    func mstart1() {
        // 通过 fs 获取 tls 数据,也就是 g0. 关于 tls 详细内容,可以参考 github.com/WANGgbin/linux_learn/syscall/tls.md.
        // 我们可以看到是如何将数据传递给新建的 m 的,通过 tls 的方式.
        _g_ := getg()
        // 一些必要的初始化
        asminit()
        // 信号相关的初始化操作
        minit()

        // 尝试为线程绑定 p
        if _g_.m != &m0 {
            acquirep(_g_.m.nextp.ptr())
            _g_.m.nextp = 0
        }

        // 调度获取 g 并运行
        schedule()
    }

    func acquirep(_p_ *p) {
        _g_.m.p.set(_p_)
        _p_.m.set(_g_.m)
        _p_.status = _Prunning
    }
    ```
    至此, 新的 m 便创建出来并绑定 p,最后执行 schedule() 函数不断的获取 goroutine 并运行.

    - m 数量限制

    m 的数量限制在 schedinit() 中作出了限制,最大为 10000.
    ```go
    sched.maxmcount = 10000
    ```

    ```go
    // startm() 中会调用此函数给新的线程分配 id
    func mReserveID() int64{
        id := sched.mnext
        sched.mnext++
        checkmcount()
        return id
    }

    func checkmcount() {
        if mcount() > sched.maxmcount {
            print("runtime: program exceeds", sched.maxmcount, "-thread limit\n")
            throw("thread exhaustion")
        }
    }
    ```
    - m 休眠如何实现

    go runtime 中的线程休眠都是通过 futex 实现的. 关于 futex 的详细描述可以参考: github.com/WANGgbin/linux_learn/syscall/futex.md.

    go runtime 中通过调用 `stopm` 阻塞一个 m, 我们看看此函数的实现:
    ```go
    func stopm() {
        _g_ := getg()
        lock(&sched.lock)
        // 将 m 扔到空闲列表中
        mput(_g_.m)
        unlock(&sched.lock)
        // 阻塞
        mPark()
        // 重新唤醒后,绑定 nextp 指定的 p
        acquirep(_g_.m.nextp.ptr())
        _g_.m.nextp = 0
    }

    func mPark() {
        g := getg()
        for {
            // park 是 note 类型, note 本质就是个 val
            notesleep(&g.m.park)
            // 清理 m.park == 0
            noteclear(&g.m.park)
            if !mDoFixup() {
                return
            }
        }
    } 
    
    func notesleep(n *note) {
        gp := getg()

        // -1 表示永久阻塞
        ns := int64(-1)
        for atomic.Load(key32(&n.key)) == 0 {
            gp.m.blocked = true
            // 如果 note.key == 0 则永久阻塞
            futexsleep(key32(&n.key), 0, ns)
            gp.m.blocked = false
        }
    }

    // 那么 m 又是如何被唤醒的呢? 通过 notewakeup

    func notewakeup(n *note) {
        // 将 note.key 设置为 1
        old := atomic.Xchg(key32(&n.key), 1)
        // 第二个参数表示唤醒几个 m
        futexwakeup(key32(&n.key), 1)
    }
    ```
- g

g 就是 goroutine.怎么理解 g 呢? g 跟 m 的区别是什么呢?
因为 m 涉及到跟 os 的交互,包括线程创建,休眠等等,会导致频繁的从用户态与内核态之间的切换. 相比与 m, g 完全是用户态的执行单元, g 的创建,休眠,销毁都是在用户态运行的.从而可以避免不必要的切换指令.


那么 g 的结构是什么样的呢? 是如何创建,调度,销毁的呢? m 又是如何调度 g 的呢?


任何一个执行流,必须要有自己的栈和 PC, 栈通常通过 sp 来表示. pc 则表示当前执行指令的地址.此外,还应该有绑定的入参. 我们来看看 g 的定义:
```go
// 删除一些字段,我们重点关注跟 g 运行,调度相关的字段.
type g struct {
	// Stack parameters.
	// stack describes the actual stack memory: [stack.lo, stack.hi).
	// stackguard0 is the stack pointer compared in the Go stack growth prologue.
	// It is stack.lo+StackGuard normally, but can be StackPreempt to trigger a preemption.
	stack       stack   // g 的栈,创建 g 的时候分配
	stackguard0 uintptr // g 的同步抢占(preempt)就是通过该字段实现的

	m         *m      // 绑定的 m 
	sched     gobuf // 调度相关. 主要就是 pc, sp 等信息

	atomicstatus uint32 // g 的状态
	goid         int64  // g 全局唯一 id
	schedlink    guintptr // goroutine 队列
	waitsince    int64      // approx time when the g become blocked
	waitreason   waitReason // if status==Gwaiting

	preempt       bool // preemption signal, duplicates stackguard0 = stackpreempt

	sysexitticks   int64    // cputicks when syscall has returned (for tracing)
	lockedm        muintptr
	startpc        uintptr         // goroutine 函数地址
	timer          *timer         // 在介绍 go 的 timer 的时候再介绍该字段
}

// 协程栈
type stack struct {
    lo uintptr
    hi uintptr
}

// 调度保存信息
type gobuf struct {
    sp   uintptr  // 栈顶
	pc   uintptr  // 当前指令地址
	g    guintptr // 指向自己对应的 g 结构体
	ctxt unsafe.Pointer // 指向 goroutine 对应的闭包, 创建 go 的是 newproc 函数, 该函数会接受一个闭包
	bp   uintptr // 栈基地址
}
```

goruntime 是通过 `newproc` 来创建协程的.我们看看该函数的实现.
```go
// siz: 入参大小
// fn: 指向闭包对象指针. funcval 就是闭包.
func newproc(siz int32, fn *funcval) {
    // 函数参数的起始地址. 在调用 newproc 之前,会现将fn 的参数复制到栈中,再调用 newproc. 
    // 所以此时的栈结构为:
    /*
    | argn |
    | ...  |
    | arg1 | <-- argp
    | fn   |
    | size |
    */
    // 通过 argp + size 即可获取所有的参数
    argp := add(unsafe.Pointer(&fn), sys.PtrSize)
	gp := getg()
	systemstack(func() {
        // g 的初始化操作在 newproc1 中完成
		newg := newproc1(fn, argp, siz, gp, pc)
        // 创建完成后, 将 g 加入到当前 p 的队列
		_p_ := getg().m.p.ptr()
		runqput(_p_, newg, true)
        
        //前面介绍过,还会尝试拉起新的线程
		if mainStarted {
			wakep()
		}
	})
}

func newproc1(fn *funcval, argp unsafe.Pointer, narg int32, callergp *g, callerpc uintptr) *g {
	_g_ := getg()

	siz := narg
    // 注意这种写法, 向上取最近的 8 的整数倍
	siz = (siz + 7) &^ 7

	_p_ := _g_.m.p.ptr()
    // 尝试从当前 p 的 g 缓存中获取空闲的 g
	newg := gfget(_p_)
	if newg == nil {
        // 获取不到的话,则通过 malg 分配.
		newg = malg(_StackMin)
		casgstatus(newg, _Gidle, _Gdead)
		allgadd(newg) // publishes with a g->status of Gdead so GC scanner doesn't look at uninitialized stack.
	}

	totalSize := 4*sys.PtrSize + uintptr(siz) + sys.MinFrameSize // extra space in case of reads slightly beyond frame
	totalSize += -totalSize & (sys.StackAlign - 1)               // align to StackAlign
	sp := newg.stack.hi - totalSize
	spArg := sp

	if narg > 0 {
        // 拷贝参数到协程栈上
		memmove(unsafe.Pointer(spArg), argp, uintptr(narg))
	}

	memclrNoHeapPointers(unsafe.Pointer(&newg.sched), unsafe.Sizeof(newg.sched))
	newg.sched.sp = sp
	newg.stktopsp = sp
    // pc 指向 goexit 函数的第二行指令
	newg.sched.pc = abi.FuncPCABI0(goexit) + sys.PCQuantum // +PCQuantum so that previous instruction is in same function
	newg.sched.g = guintptr(unsafe.Pointer(newg))
    // 这个函数很重要,会更新 newg.sched, 模拟 go.exit 调用 fn 的场景. 为什么要模拟这个场景呢? 因为这样的话,当 fn 执行完毕后,就可以执行 go 正常的退出流程了.
	gostartcallfn(&newg.sched, fn)
	newg.gopc = callerpc
	newg.ancestors = saveAncestors(callergp)
	newg.startpc = fn.fn

	// 调整状态为 _Grunnable
	casgstatus(newg, _Gdead, _Grunnable)

	// 设置协程 id
	newg.goid = int64(_p_.goidcache)
	_p_.goidcache++

	return newg
}

// gostartcallfn 内部主要调用 gostartcall()
func gostartcall(buf *gobuf, fn, ctxt unsafe.Pointer) {
    // buf.pc 是指向 goexit 的第二条指令
    // 栈顶插入 buf.pc
	sp := buf.sp
	sp -= sys.PtrSize
	*(*uintptr)(unsafe.Pointer(sp)) = buf.pc
	buf.sp = sp
    // pc 指向 fn 对应的函数. 当 fn 执行完毕之后, 就开始从 goexit 第二行开始执行
	buf.pc = uintptr(fn)
	buf.ctxt = ctxt
}
```

以上便是 goroutine 的整个创建流程. 我们来看看协程退出的时候发生了什么.前面讲过, 协程的推出操作是在 goexit 中完成的.我们来看看该函数的实现.
```go
TEXT runtime·goexit(SB),NOSPLIT|TOPFRAME,$0-0
	BYTE	$0x90	// NOP
	CALL	runtime·goexit1(SB)	// does not return
	// traceback from goexit1 must hit code range of goexit
	BYTE	$0x90	// NOP

// 函数内部主要调用 goexit1() 函数
func goexit1() {
	mcall(goexit0)
}

// 注意此函数是在 m 的线程栈上运行的
func goexit0(gp *g) {
	_g_ := getg()

    // 状态设置为 _Gdead
	casgstatus(gp, _Grunning, _Gdead)
	if isSystemGoroutine(gp, false) {
		atomic.Xadd(&sched.ngsys, -1)
	}
    // 清理 g 的各项成员,注意栈是仍然保留的,不然缓存就没意义了.
	gp.m = nil
	locked := gp.lockedm != 0
	gp.lockedm = 0
	_g_.m.lockedg = 0
	gp.preemptStop = false
	gp.paniconfault = false
	gp._defer = nil // should be true already but just in case.
	gp._panic = nil // non-nil for Goexit during panic. points at stack-allocated data.
	gp.writebuf = nil
	gp.waitreason = 0
	gp.param = nil
	gp.labels = nil
	gp.timer = nil

    // 解除 g 跟 m 的绑定关系
	dropg()
    // 扔到 p 的 g 空闲链表中,方便之后复用
	gfput(_g_.m.p.ptr(), gp)
    // 调度执行新的 g, 此函数从不返回
	schedule()
}
```

接下来,我们看看最重要的 schedule(), m 究竟是如何寻找可执行的 goroutine 的呢? schedule() 的大体思路是:
- 从当前 p 的 runnalbe 队列中寻找可执行 g
- 从全局 runnable 队列中寻找可执行 g
- 从其他 p 的 runnable 队列中寻找可执行 g
- 尝试执行网络 i/o,或者 timer 唤醒阻塞的 g
- 
```go
func schedule() {
	_g_ := getg()

top:
	pp := _g_.m.p.ptr()
	pp.preempt = false

    // 与 STW 相关,后面在 GC 部分再分析
	if sched.gcwaiting != 0 {
		gcstopm()
		goto top
	}

    // 处理当前 p 的定时器,后面分析 timers 的时候会详细分析
	checkTimers(pp, 0)

	var gp *g
	var inheritTime bool

	// Normal goroutines will check for need to wakeP in ready,
	// but GCworkers and tracereaders will not, so the check must
	// be done here instead.
	tryWakeP := false

    // 检测 traceReader goroutine
	if trace.enabled || trace.shutdown {
		gp = traceReader()
		if gp != nil {
			casgstatus(gp, _Gwaiting, _Grunnable)
			traceGoUnpark(gp, 0)
			tryWakeP = true
		}
	}
    // 检测 gcWorker
	if gp == nil && gcBlackenEnabled != 0 {
		gp = gcController.findRunnableGCWorker(_g_.m.p.ptr())
		if gp != nil {
			tryWakeP = true
		}
	}
    // 以一定的概率从全局队列获取 g
	if gp == nil {
		// Check the global runnable queue once in a while to ensure fairness.
		// Otherwise two goroutines can completely occupy the local runqueue
		// by constantly respawning each other.
		if _g_.m.p.ptr().schedtick%61 == 0 && sched.runqsize > 0 {
			lock(&sched.lock)
			gp = globrunqget(_g_.m.p.ptr(), 1)
			unlock(&sched.lock)
		}
	}
    // 从当前 p 的队列中获取 g
	if gp == nil {
		gp, inheritTime = runqget(_g_.m.p.ptr())
		// We can see gp != nil here even if the M is spinning,
		// if checkTimers added a local goroutine via goready.
	}

    // 调用 findrunnable() 获取 g
	if gp == nil {
		gp, inheritTime = findrunnable() // blocks until work is available
	}

	// This thread is going to run a goroutine and is not spinning anymore,
	// so if it was marked as spinning we need to reset it now and potentially
	// start a new spinning M.
	if _g_.m.spinning {
		resetspinning()
	}

    // 如果当前禁止调度用户协程,则将协程加入到 disable 队列中.当允许调度用户线程的时候, disable 队列中的 goroutine 会加入到 runq 中.
    // 什么时候会禁止调度用户协程呢?跟 GC 有关吗?
	if sched.disable.user && !schedEnabled(gp) {
		// Scheduling of this goroutine is disabled. Put it on
		// the list of pending runnable goroutines for when we
		// re-enable user scheduling and look again.
		lock(&sched.lock)
		if schedEnabled(gp) {
			// Something re-enabled scheduling while we
			// were acquiring the lock.
			unlock(&sched.lock)
		} else {
			sched.disable.runnable.pushBack(gp)
			sched.disable.n++
			unlock(&sched.lock)
			goto top
		}
	}

	// If about to schedule a not-normal goroutine (a GCworker or tracereader),
	// wake a P if there is one.
	if tryWakeP {
		wakep()
	}
    // 如果 gp 有绑定的 m 则唤醒 lockedm 并把 p hand off 给 lockm. m 睡眠等待被重新唤醒,然后从 top 开始重新执行
	if gp.lockedm != 0 {
		// Hands off own p to the locked m,
		// then blocks waiting for a new p.
		startlockedm(gp)
		goto top
	}

    // 内部通过调用 gogo 函数切换到 gp
	execute(gp, inheritTime)
}

// 我们看看 findrunnable() 函数的实现
func findrunnable() (gp *g, inheritTime bool) {
	_g_ := getg()

top:
	_p_ := _g_.m.p.ptr()
	if sched.gcwaiting != 0 {
		gcstopm()
		goto top
	}
	if _p_.runSafePointFn != 0 {
		runSafePointFn()
	}

    // 通过定时器尝试激活一些 g
	now, pollUntil, _ := checkTimers(_p_, 0)

	// local runq
	if gp, inheritTime := runqget(_p_); gp != nil {
		return gp, inheritTime
	}

	// global runq
	if sched.runqsize != 0 {
		lock(&sched.lock)
		gp := globrunqget(_p_, 0)
		unlock(&sched.lock)
		if gp != nil {
			return gp, false
		}
	}

	// Poll network.
	// This netpoll is only an optimization before we resort to stealing.
	// We can safely skip it if there are no waiters or a thread is blocked
	// in netpoll already. If there is any kind of logical race with that
	// blocked thread (e.g. it has already returned from netpoll, but does
	// not set lastpoll yet), this thread will do blocking netpoll below
	// anyway.
    // 尝试通过 netpoll 激活阻塞在网络 io 的协程
	if netpollinited() && atomic.Load(&netpollWaiters) > 0 && atomic.Load64(&sched.lastpoll) != 0 {
        // list 为激活的协程, 0 表示非阻塞. 在 linux+x86 上, netpoll 底层调用的就是 epoll
		if list := netpoll(0); !list.empty() { // non-blocking
			// 获取第一个剩下的加入到 runq 中
            gp := list.pop()
			injectglist(&list)
			casgstatus(gp, _Gwaiting, _Grunnable)
			return gp, false
		}
	}

	// Spinning Ms: steal work from other Ps.
	//
	// Limit the number of spinning Ms to half the number of busy Ps.
	// This is necessary to prevent excessive CPU consumption when
	// GOMAXPROCS>>1 but the program parallelism is low.
	procs := uint32(gomaxprocs)
	if _g_.m.spinning || 2*atomic.Load(&sched.nmspinning) < procs-atomic.Load(&sched.npidle) {
		if !_g_.m.spinning {
			_g_.m.spinning = true
			atomic.Xadd(&sched.nmspinning, 1)
		}

        // 从其他 p 偷取 g
		gp, inheritTime, tnow, w, newWork := stealWork(now)
		now = tnow
		if gp != nil {
			// Successfully stole.
			return gp, inheritTime
		}
		if newWork {
			// There may be new timer or GC work; restart to
			// discover.
			goto top
		}
		if w != 0 && (pollUntil == 0 || w < pollUntil) {
			// Earlier timer to wait for.
			pollUntil = w
		}
	}

	// We have nothing to do.
	//
	// If we're in the GC mark phase, can safely scan and blacken objects,
	// and have work to do, run idle-time marking rather than give up the
	// P.
    // 如果有 gcMarkWork 可用,则返回 gcMarkWork
	if gcBlackenEnabled != 0 && gcMarkWorkAvailable(_p_) {
		node := (*gcBgMarkWorkerNode)(gcBgMarkWorkerPool.pop())
		if node != nil {
			_p_.gcMarkWorkerMode = gcMarkWorkerIdleMode
			gp := node.gp.ptr()
			casgstatus(gp, _Gwaiting, _Grunnable)
			return gp, false
		}
	}

	// 释放 p 并将 p 加入到 idle 队列
	if releasep() != _p_ {
		throw("findrunnable: wrong p")
	}
	pidleput(_p_)
	unlock(&sched.lock)

    // m 睡眠, 当新建 g 或者 g ready 的时候,激活 m
	stopm()
	goto top
}

// 我们看看 startlockedm() 函数的实现
func startlockedm(gp *g) {
	_g_ := getg()

	mp := gp.lockedm.ptr()

	// directly handoff current P to the locked m
	incidlelocked(-1)
    // 让出 p
	_p_ := releasep()
    // p 跟 lockm 绑定
	mp.nextp.set(_p_)
    // 激活 lockm
	notewakeup(&mp.park)
    // m 阻塞
	stopm()
}
```

我们接着看看 execute() 是如何执行一个 g 的. 本质上就是使用 g.sched 恢复上下文.

```go
func execute(gp *g, inheritTime bool) {
	_g_ := getg()

	// gp 跟 m 互相绑定
	_g_.m.curg = gp
	gp.m = _g_.m
    // 状态切换为 _Grunning
	casgstatus(gp, _Grunnable, _Grunning)
	gp.waitsince = 0
	gp.preempt = false
	gp.stackguard0 = gp.stack.lo + _StackGuard
	if !inheritTime {
		_g_.m.p.ptr().schedtick++
	}
    // store gp 上下文
	gogo(&gp.sched)
}

TEXT runtime·gogo(SB), NOSPLIT, $0-8
	MOVQ	buf+0(FP), BX		// gobuf
	MOVQ	gobuf_g(BX), DX
	MOVQ	0(DX), CX		// make sure g != nil
	JMP	gogo<>(SB)

// gogo 主要做了两件事情,一个是更新 tls 为当前 g, 另一个就是 load g
TEXT gogo<>(SB), NOSPLIT, $0
	get_tls(CX)
	MOVQ	DX, g(CX) // 更新 tls 为当前的 g
	MOVQ	DX, R14		
	MOVQ	gobuf_sp(BX), SP	// restore SP
	MOVQ	gobuf_ret(BX), AX
	MOVQ	gobuf_ctxt(BX), DX // 实际上,函数内部就是通过 DX 来访问闭包变量的
	MOVQ	gobuf_bp(BX), BP
	MOVQ	$0, gobuf_sp(BX)	// clear to help garbage collector ????
	MOVQ	$0, gobuf_ret(BX)
	MOVQ	$0, gobuf_ctxt(BX)
	MOVQ	$0, gobuf_bp(BX)
	MOVQ	gobuf_pc(BX), BX // 永远是最后一步
	JMP	BX // 跳转到 gobuf_pc(BX)
```
- g 的抢占

g 是如何被抢占的呢? 这里的抢占值得是,如果当前 g 时间片到期,如何通知其让出呢?

在 go 1.13 之前,是同步抢占. 即将 g.stackguard0 设置为 stackPreempt, 然后 g 在栈增长检测代码中检查该标志,让出并调用 schedule(). 这种抢占模型,在 for{} 死循环下会有问题,因为 for{} 并不会有栈增长检测.

从 go 1.14 开始,真正实现了异步抢占.这里的异步同步是站在被抢占的 g 的角度考虑的. 通过发送一个信号, 在信号处理函数中更改用户态 pc, 在信号处理函数返回后,执行调度.我们看看信号处理函数的实现:
```go
// runtime.sighandler() 否则处理信号
// ctxt 表示保存用户态寄存器堆栈起始地址
func sighandler(sig uint32, info *siginfo, ctxt unsafe.Pointer, gp *g) {
    if sig == sigPreempt && debug.asyncpreemptoff == 0 {
		// Might be a preemption signal.
		doSigPreempt(gp, c)
	}
}

// doSigPreempt handles a preemption signal on gp.
func doSigPreempt(gp *g, ctxt *sigctxt) {
	// Check if this G wants to be preempted and is safe to
	// preempt.
	if wantAsyncPreempt(gp) {
		if ok, newpc := isAsyncSafePoint(gp, ctxt.sigpc(), ctxt.sigsp(), ctxt.siglr()); ok {
			// Adjust the PC and inject a call to asyncPreempt.
            // 这是最关键的一步:设置 pc, sp, 模拟调用 asyncPreempt 场景.
            // 我们已经在 runtime 里面看到很多使用这种技术的场景了.
			ctxt.pushCall(funcPC(asyncPreempt), newpc)
		}
	}
}

// 下面是信号处理函数 ctxt 结构的解析
func (c *sigctxt) sigpc() uintptr { return uintptr(c.rip()) }
func (c *sigctxt) rip() uint64 { return c.regs().rip }
func (c *sigctxt) regs() *sigcontext {
	return (*sigcontext)(unsafe.Pointer(&(*ucontext)(c.ctxt).uc_mcontext))
}

type ucontext struct {
	uc_flags     uint64
	uc_link      *ucontext
	uc_stack     stackt
	uc_mcontext  mcontext
	uc_sigmask   usigset
	__fpregs_mem fpstate
}

type sigcontext struct {
	r8          uint64
	r9          uint64
	r10         uint64
	r11         uint64
	r12         uint64
	r13         uint64
	r14         uint64
	r15         uint64
	rdi         uint64
	rsi         uint64
	rbp         uint64
	rbx         uint64
	rdx         uint64
	rax         uint64
	rcx         uint64
	rsp         uint64
	rip         uint64
	eflags      uint64
	cs          uint16
	gs          uint16
	fs          uint16
	__pad0      uint16
	err         uint64
	trapno      uint64
	oldmask     uint64
	cr2         uint64
	fpstate     *fpstate1
	__reserved1 [8]uint64
}

type mcontext struct {
	gregs       [23]uint64  // 与上面的 sigcontext 完全一致
	fpregs      *fpstate
	__reserved1 [8]uint64
}

// 我们接着看看真正执行异步抢占函数的实现

// asyncPreempt 首先保存所有寄存器,然后调用 asyncPreempt2() 函数, 从 asyncPreempt2() 返回后,恢复上下文. 从 asyncPreempt() 返回后,回到被信号中断的上下文(这是在ctxt.pushCall中完成的),继续执行.
func asyncPreempt() // 通过汇编实现
func asyncPreempt2() {
	gp := getg()
	gp.asyncSafePoint = true
    // gp 上下文保存是在 mcall 中执行的. mcall 就是在系统栈上执行函数,并且永远不会切回到协程栈上. 当被保存的协程再次执行的时候,就是从 mcall() 后面开始执行的.
	if gp.preemptStop {
		mcall(preemptPark)
	} else {
		mcall(gopreempt_m)
	}
	gp.asyncSafePoint = false
}

// gopreempt_m 内部调用 goschedImpl 函数
func goschedImpl(gp *g) {
	status := readgstatus(gp)
    // 更改状态
	casgstatus(gp, _Grunning, _Grunnable)
	dropg()
    // 加入到 runq
	lock(&sched.lock)
	globrunqput(gp)
	unlock(&sched.lock)
    // 调度
	schedule()
}
```
总体来说,异步抢占式调度有这个几个关键点:
 - 什么时候执行抢占操作呢? 通过在信号处理函数中 pushCall 的方式,使得从信号处理函数返回后, 执行pushCall 指定的操作而不是延续之前的上下文.
 - 被调度的 g 要能继续从之前的上下文执行,因此 pushCall 是模拟的调用 asyncPreemt 场景,而不是直接将 pc 设置为 asyncPreemt.
 - 因为是异步中断,所以要保存上下文(所有寄存器)
 - 被调度的 g 的 gobuf 是在 mcall() 保存的,保证下次调度执行的时候,能够进跟着 mcall() 继续执行.

怎么理解 mcall 函数呢?
个人理解 mcall 就是**用于 g 的调度的**. 比如要让出一个 g, g 的sp, pc 应该保存为什么值呢? mcall() 提供了一个清晰的边界,保存为 mcall()下一条指令. 接下来就是执行一些操作以及 schedule() 获取可执行的 g,因为线程系统栈足够大,在 系统栈上运行这部分函数. 当找到可运行的 g 后,又通过 executor() 切换到对应的 g.

**在 go runtime 中,我们可以看到使用 mcall() 的地方都跟调度有关系**. mcall() 就是调度 g 上下文保存恢复的标准. 为什么 mcall() 不返回到原来的 g, 因为 mcall() 就是用于 g 调度的.

systemstack() 跟 mcall() 很相似. 都是切换到系统栈上运行函数. 但是 systemstack() 的定位就是因为一些频繁调用的函数会涉及栈增长,直接在 g 栈上运行这些函数会造成一些不必要的栈增长.因此将这部分 runtime 中的高频函数的执行收敛到系统栈. 因为压根跟调度没关系,纯粹是为了切换到系统栈执行函数,所以systemstack() 执行完函数会恢复到原来的 g,


这里有个问题,主动让出,比如睡眠,阻塞为什么不保存 cpu 上下文呢?
- p

什么是 p 呢? 为什么要有 p?
p 可以理解为是一个资源.一个 m 要运行,必须要绑定 p 之后才可以运行. 个人认为 p 的引入有两个好处:
 - 资源分配

资源划分到 p 粒度, 减少竞争,从而提高性能
 - 并行控制

保证最多同时有 GOMAXPROCS 个线程运行. 最大程度利用多核性能,避免线程太少或者太多.

- 系统调用

系统调用部分可以参考 syscall,
