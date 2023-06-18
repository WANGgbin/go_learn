描述 go runtime 的网络模型。

# 总体思路

go 的 runtime 封装了网络/磁盘 IO, 应用层跟 runtime 层交互。 应用层进行 I/O 的时候，首先进行一次无阻塞
操作，如果操作失败，则协程阻塞在相应的 fd 上。

runtime 会在几个地方不时的调用 netpoll 函数，如果某个套接字可读写，则会激活阻塞在该套接字上的协程。

# 详细设计

## 核心数据结构

### poll.FD

不管是套接字还是普通的文件描述符，都会引用该结构。

```go
type FD struct {
	// Lock sysfd and serialize access to Read and Write methods.
	fdmu fdMutex

	// System file descriptor. Immutable until Close.
	Sysfd int

	// I/O poller.
	pd pollDesc

	// Writev cache.
	iovecs *[]syscall.Iovec

	// Semaphore signaled when file is closed.
	csema uint32

	// Non-zero if this file has been set to blocking mode.
	isBlocking uint32

	// Whether this is a streaming descriptor, as opposed to a
	// packet-based descriptor like a UDP socket. Immutable.
	IsStream bool

	// Whether a zero byte read indicates EOF. This is false for a
	// message based socket connection.
	ZeroReadIsEOF bool

	// Whether this is a file rather than a network socket.
	isFile bool
}
```

FD 中有个很重要的成员 `pd`， FD 的读写操作都是基于 pd 来完成的。这也是应用层跟 runtime 的边界所在。

我们看看 FD 读操作的实现，其他操作类似，不再描述。

```go
func (fd *FD) Read(p []byte) (int, error) {
	if err := fd.readLock(); err != nil {
		return 0, err
	}
	defer fd.readUnlock()
	if err := fd.pd.prepareRead(fd.isFile); err != nil {
		return 0, err
	}
	if fd.IsStream && len(p) > maxRW {
		p = p[:maxRW]
	}
	for {
		// 进行非阻塞读，套接字在创建的时候，会通过 fcntl 设置 O_NONBLOCK.
		n, err := ignoringEINTRIO(syscall.Read, fd.Sysfd, p)
		if err != nil {
			n = 0
			// 非阻塞读如果没有数据返回 EAGAIN/EWOULDBLOCK 错误
			if err == syscall.EAGAIN && fd.pd.pollable() {
				// 协程挂起，都再次可读的时候，waitRead() 返回
				if err = fd.pd.waitRead(fd.isFile); err == nil {
					continue
				}
			}
		}
		err = fd.eofError(n, err)
		return n, err
	}
}
```

### runtime.pollDesc

协程阻塞的时候就是挂在 pollDesc 这个资源上的，同时该结构体也被注入到 event.data 中。其定义为：
```go
type pollDesc struct {
	link *pollDesc // in pollcache, protected by pollcache.lock

	// The lock protects pollOpen, pollSetDeadline, pollUnblock and deadlineimpl operations.
	// This fully covers seq, rt and wt variables. fd is constant throughout the PollDesc lifetime.
	// pollReset, pollWait, pollWaitCanceled and runtime·netpollready (IO readiness notification)
	// proceed w/o taking the lock. So closing, everr, rg, rd, wg and wd are manipulated
	// in a lock-free way by all operations.
	// NOTE(dvyukov): the following code uses uintptr to store *g (rg/wg),
	// that will blow up when GC starts moving objects.
	lock    mutex // protects the following fields
	fd      uintptr
	closing bool
	everr   bool      // marks event scanning error happened
	user    uint32    // user settable cookie
	rseq    uintptr   // protects from stale read timers
	rg      uintptr   // pdReady, pdWait, G waiting for read or nil
	rt      timer     // read deadline timer (set if rt.f != nil)
	rd      int64     // read deadline
	wseq    uintptr   // protects from stale write timers
	wg      uintptr   // pdReady, pdWait, G waiting for write or nil
	wt      timer     // write deadline timer
	wd      int64     // write deadline
	self    *pollDesc // storage for indirect interface. See (*pollDesc).makeArg.
}
```

waitRead() 底层调用的是 poll_runtime_pollWait() 函数，我们看看该函数的实现，只保留了主体部分。

```go
func poll_runtime_pollWait(pd *pollDesc, mode int) int {
	// netpollblock 调用 gopark 阻塞协程，一致等到协程再次可运行.
	// 当可读写的时候，函数返回 true
	// 超时或者套接字关闭的时候，函数返回 false
    for !netpollblock(pd, int32(mode), false) {
        errcode = netpollcheckerr(pd, int32(mode))
        if errcode != pollNoError {
            return errcode
        }
        // Can happen if timeout has fired and unblocked us,
        // but before we had a chance to run, timeout has been reset.
        // Pretend it has not happened and retry.
        }
    return pollNoError
}
```

那么阻塞的协程由谁激活呢？netpoll 函数负责激活协程，负责检查网络连接，并返回可运行的协程列表。
在调度器、监视线程等场景中都会调用该函数。因为监视线程不在 GMP 调度模型中，因此 delay > 0, 其他场景
中都是非阻塞的调用 epoll_wait.

我们简单看看 netpoll 的实现：
```go
// netpoll checks for ready network connections.
// Returns list of goroutines that become runnable.
// delay < 0: blocks indefinitely
// delay == 0: does not block, just polls
// delay > 0: block for up to that many nanoseconds
func netpoll(delay int64) gList {
	var events [128]epollevent
retry:
	// linux 平台通过 epollwait 完成套接字检查
	n := epollwait(epfd, &events[0], int32(len(events)), waitms)
	if n < 0 {
		if n != -_EINTR {
			println("runtime: epollwait on fd", epfd, "failed with", -n)
			throw("runtime: netpoll failed")
		}
		// If a timed sleep was interrupted, just return to
		// recalculate how long we should sleep now.
		if waitms > 0 {
			return gList{}
		}
		goto retry
	}
	var toRun gList
	for i := int32(0); i < n; i++ {
		ev := &events[i]
		if ev.events == 0 {
			continue
		}
        
		// 判断读写事件
		var mode int32
		if ev.events&(_EPOLLIN|_EPOLLRDHUP|_EPOLLHUP|_EPOLLERR) != 0 {
			mode += 'r'
		}
		if ev.events&(_EPOLLOUT|_EPOLLHUP|_EPOLLERR) != 0 {
			mode += 'w'
		}
		if mode != 0 {
			pd := *(**pollDesc)(unsafe.Pointer(&ev.data))
			pd.everr = false
			if ev.events == _EPOLLERR {
				pd.everr = true
			}
			// 调用 netpollready 将激活的协程添加到 toRun 中
			netpollready(&toRun, pd, mode)
		}
	}
	return toRun
}
```

另一个问题是，读写的超时是如何实现的呢？通过 runtime 的 timer 来完成。
当设置套接字的读写超时事件的时候，实际上是在 runtime 的 timer heap 中创建了一个 timer.

执行路径为：
FD.SetDeadline()
    -> FD.setDeadlineImpl()
        -> runtime.poll_runtime_pollSetDeadline()

函数 runtime.poll_runtime_pollSetDeadline() 会创建一个 timer， 
timer 关联的函数为：netpollReadDeadline/netpollWriteDeadline/netpollDeadline。
3 个函数都基于 netpolldeadlineimpl。

我们看看 netpolldeadlineimpl 的实现：
```go
func netpolldeadlineimpl(pd *pollDesc, seq uintptr, read, write bool) {
	lock(&pd.lock)
	// Seq arg is seq when the timer was set.
	// If it's stale(不新鲜的), ignore the timer event.
	
	// seq 是干什么的？
	// 一个套接字上可以多次设置定时器，旧的定时器不应该影响套接字。
	// seq 就是跟定时器关联的，没创建一个定时器，seq++
	// 因此，当 pd.rseq != seq 的时候，直接忽略该定时器。
	// rseq 用于读定时器，wseq 用于写定时器。
	currentSeq := pd.rseq
	if !read {
		currentSeq = pd.wseq
	}
	if seq != currentSeq {
		// The descriptor was reused or timers were reset.
		unlock(&pd.lock)
		return
	}
	var rg *g
	if read {
		if pd.rd <= 0 || pd.rt.f == nil {
			throw("runtime: inconsistent read deadline")
		}
		pd.rd = -1
		// 激活阻塞的协程，套接字可能在超时前变的可读，因此 rg 可能为 nil
		rg = netpollunblock(pd, 'r', false)
	}
	var wg *g
	if write {
		if pd.wd <= 0 || pd.wt.f == nil && !read {
			throw("runtime: inconsistent write deadline")
		}
		pd.wd = -1
		wg = netpollunblock(pd, 'w', false)
	}
	unlock(&pd.lock)
	// 如果不为空，则表示激活协程，goready 修改协程状态并扔到当前 p 对应的运行队列上。
	if rg != nil {
		netpollgoready(rg, 0)
	}
	if wg != nil {
		netpollgoready(wg, 0)
	}
}
```
