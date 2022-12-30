描述 go 中 syscall 的实现以及跟 gmp 调度模型相关内容.
关于 go 的系统调用可以参考: [系统调用](https://xargin.com/syscall/)

# 实现
go 的 syscall包 本质上就是通过汇编指令SYSCALL 陷入内核. 内核系统调用之返回一个整型,如果系统调用出错,则返回一个表示具体错误信息的负值. syscall 包在系统调用出错的时候,会
将负值包装为一个 err 并返回(定义 Errno 类型,并实现 error 接口).

在 c 的标准库中,会将错误吗赋值到 errno 全局变量中. 本质上不管是 go 还是 c 调用系统
调用的方式都是一样的.

这里简单注意下 linux+x86 上系统调用参数传递的约定,在 x86 中,系统调用参数是通过寄存器传递的.
rax: 传递系统调用号
rdi: 第一个参数
rsi: 第二个参数
rdx: 第三个参数
r10: 第四个参数
r9: 第五个参数
r8: 第六个参数

# 如何跟 gmp 交互
因为调用系统调用,是有可能导致所在的线程阻塞的. 因此在陷入内核前需要让出 p, 从而保证其他的 m 可以获取此 p 并执行任务. 当线程从系统调用返回后,则需要重新尝试绑定之前的 p, 如果绑定不了,则将当前的 g 扔到 runq 中,当前的线程阻塞,扔到空闲列表中. 系统调用前后的这两不分别是在 `entersyscall` 和 `exitsyscall` 中实现的.我们来看看函数的具体实现.

- entersyscall

函数主要就是解绑 p 跟 m, 并更改 p 和 g 的状态.
```go
func reentersyscall(pc, sp uintptr) {
	_g_ := getg()

	_g_.stackguard0 = stackPreempt
	_g_.throwsplit = true

	// Leave SP around for GC and traceback.
	save(pc, sp)
	_g_.syscallsp = sp
	_g_.syscallpc = pc
    // 将 g 状态设置为 _Gsyscall
	casgstatus(_g_, _Grunning, _Gsyscall)

	_g_.m.syscalltick = _g_.m.p.ptr().syscalltick
	_g_.sysblocktraced = true
	// 解绑 p 跟 m
    pp := _g_.m.p.ptr()
	pp.m = 0
    // 将 m.oldp 设置为 pp, 后续在 exitsyscall 中绑定 p 的时候会用到
	_g_.m.oldp.set(pp)
	_g_.m.p = 0
    // 设置 p 状态为 _Psyscall
	atomic.Store(&pp.status, _Psyscall)
}
```
- exitsyscall

exitsyscall 主要就是尝试绑定原来的 p, 如果绑定不了则绑定一个空闲的p,如果都不行的话,则将 g 扔到全局 runq, m 阻塞.
```go
func exitsyscall() {
	_g_ := getg()
	oldp := _g_.m.oldp.ptr()
	_g_.m.oldp = 0
    // 如果可以直接绑定 oldp, 则直接返回
	if exitsyscallfast(oldp) {
		// There's a cpu for us, so we can run.
		// We need to cas the status and scan before resuming...
		casgstatus(_g_, _Gsyscall, _Grunning)

		return
	}

	_g_.sysexitticks = 0

	// Call the scheduler.
	mcall(exitsyscall0)
}

func exitsyscallfast(oldp *p) bool {
	_g_ := getg()

	// Try to re-acquire the last P.
    // 尝试获取 oldp
	if oldp != nil && oldp.status == _Psyscall && atomic.Cas(&oldp.status, _Psyscall, _Pidle) {
		// oldp 可以使用, wirep 主要就是重新绑定 m 和 p
		wirep(oldp)
		return true
	}

	// Try to get any other idle P.
    // 尝试获取其他空闲的 idle
	if sched.pidle != 0 {
		var ok bool
		systemstack(func() {
			ok = exitsyscallfast_pidle() // 获取空闲 p 并 acquire
		})
		if ok {
			return true
		}
	}
	return false
}

func exitsyscall0(gp *g) {
    // 更改状态为 _Grunnable
	casgstatus(gp, _Gsyscall, _Grunnable)
	dropg()
	lock(&sched.lock)
	var _p_ *p
    // 尝试获取空闲 p
	if schedEnabled(gp) {
		_p_ = pidleget()
	}
	var locked bool
	if _p_ == nil {
        // 如果未获取则将 gp 扔到全局 runq 中,并且 stopm()
		globrunqput(gp)

		// Below, we stoplockedm if gp is locked. globrunqput releases
		// ownership of gp, so we must check if gp is locked prior to
		// committing the release by unlocking sched.lock, otherwise we
		// could race with another M transitioning gp from unlocked to
		// locked.
		locked = gp.lockedm != 0
	} else if atomic.Load(&sched.sysmonwait) != 0 {
		atomic.Store(&sched.sysmonwait, 0)
		notewakeup(&sched.sysmonnote)
	}
	unlock(&sched.lock)
    // 如果获取到 p, 则绑定 m 和 p, 并继续执行 gp
	if _p_ != nil {
		acquirep(_p_)
		execute(gp, false) // Never returns.
	}
    // 如果没有获取到 p, 则阻塞 m
	stopm()
	schedule() // Never returns.
}

```

# 系统调用可能引发的问题
如果大量的协程调用长时间阻塞的系统调用,就可能导致创建大量的 m. 当 m 的数量超过 runtime 的限制(10000)的时候,程序会 panic. 所以,一般在业务代码中,我们最好不要直接调用系统调用.而是调用 runtime 封装好的函数. 由 runtime 来负责跟 os 的交互. 比如像文件的读写, 实际上并不是直接调用系统调用.而是在 runtime 中封装了一层.