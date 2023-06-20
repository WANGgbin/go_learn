package main

/*
探索 go 的网络输入输出模型
go 的网络模型位于内核和应用层之间,既要保证应用层的易用,又要保证高性能.接下来从这两个角度来分析下 go
网络标准库的实现.

1.应用层的易用性
	要给用户提供易用的接口,就需要精心设计,封装.我们来看看 go 是如何封装套接字的 API 的.
 1. tcp 套接字
 
 2. uds(unix domain socket) 套接字
 3. udp 套接字
 4. http 封装

2.高性能
注意: 后续使用 epoll 泛指 i/o 模型

 1. i/o 模型
	经典的 epoll 模型. O(1) 时间复杂度获取可用套接字.

 2. 套接字不可读写的时候,如何实现协程维度的阻塞而不是线程维度的阻塞
	如果直接通过系统调用直接进行套接字的读写,就会导致线程的阻塞,这样就可能导致创建成千上万的线程.实际上,
	在 go 中,会异步的时不时的调用 i/o 模型,来判断某些套接字是否可读写,会异步的维护好套接字的读写状态,
	当协程读写的时候,如果可读写直接进行相应的操作,否则将协程插入到套接字的读/写队列即可,异步程序在判断
	套接字 ready 的时候,激活对应的协程.这样就实现了协程粒度的阻塞.

	那么在调用 epoll 的时候,是否会导致线程阻塞呢?如果阻塞的话也可能会导致创建大量的线程的.如果了解 epoll 的话
	实际上是可以设置等待时间的. epoll_wait 的原型如下:
	int epoll_wait (int __epfd, struct epoll_event *__events,
		       int __maxevents, int __timeout);
	__timeout 指定等待多少 ms, -1 表示阻塞, 0 表示非阻塞立即返回.

	我们看看 go 中 epoll_wait 调用中 __timeout 是如何设置的? 实际上会在多个地方调用 netpool(netpool 内部会调用不同的平台的 wait 函数,比如
	对于 epoll_wait, 可以参考: go/src/runtime/netpoll_epoll.go/netpoll) 函数,
	比如 gc,监控线程,调度器都会调用 netpool 函数,只有监控线程会设置 > 0的值,其他调用都设置为0,监控线程不属于 GPM 模型.

 3. 套接字的读写超时时间是如何实现的呢?
	通常有以下两种方式来设置套接字的读写超时时间.
	1. 通过 epoll_wait 类函数,此类函数适用于所有的描述符
	2. 通过设置套接字的 SO_RCVTIMEO 和 SO_SNDTIMEO 选项. 缺点是这两个选项只适用于套接字选项且不一定
	所有的平台都支持

	那么 go 内部也是通过这两种方式之一实现套接字的读写超时的吗?
	因为上述两种方式都会导致线程级阻塞,所以都不适合在 go 内部使用.那么 go 内部是如何实现的呢?

	go 通过自己的定时器机制实现套接字/描述符的读写超时.每次读写前都需要设置超时定时器.调用链路如下:
	net.conn.SetDeadline()
		-> net.netFD.SetDeadline()
			-> internal/poll.FD.SetDeadline()
				-> runtime.poll_runtime_pollSetDeadline()
					-> 如果之前未设置过定时器, resettimer() 设置定时器,定时器函数为: netpollDeadline
					-> 如果之前设置过定时器, 则 pd.sep++,且更改定时器(del + adjust + add)

	继续看看 netpollDeadline() 实现:
	netpollDeadline()
		-> netpolldeadlineimpl()
			-> pd.seq == timer.seq ?
				为了防止老的定时器产生影响, pollDesc 中引入序号 seq 的概念, seq 与定时器一一对应,通过判断
				pd.seq == timer.seq 即可判断当前 timer 是否是有效(最新)的定时器.
				-> 若不是,返回
				-> 若是:
					-> pd.rd = -1,通过该字段,重新激活的协程便知道是超时事件发生引起的激活.
					-> netpollunblock(,,false)

	netpollunblock() 从名字就可以看出,函数负责激活阻塞的协程.实际上,当套接字可读/可写 or 超时 or 关闭,都会调用此函数
	负责激活阻塞的协程. 后面详细分析该函数的实现.

 4. 协程读写数据底层实现流程
	net.conn.Read()
		-> net.netFD.Read()
			-> internal/poll.FD.Read()
				-> 直接调用系统调用尝试读一次: n, err := ignoringEINTRIO(syscall.Read, fd.Sysfd, p)
					-> 读到数据,直接返回
					-> 否则,调用 internal/poll.pollDesc.waitRead()
						-> runtime.poll_runtime_pollWait()
							-> netpollblock() 协程阻塞,当 i/o ready or 超时 or 关闭的时候,激活协程.
								协程激活后,通过判断 pollDesc.rg == pdReady, 来判断是由那种情况激活的.
								如果不想等,则调用 netpollcheckerr()
							-> netpollcheckerr()
								-> pollDesc.closing == true, 则套接字关闭
								-> pollDesc.rd < 0,则套接字读超时. 当定时器到达的时候,会设置 pollDesc.rd == -1

	套接字的写操作类似,不再赘述.

5. 当 i/o ready 的时候,底层流程是什么呢?
	前面描述过, go 会在几个不同的地方时不时的调用 runtime.netpoll() 函数.我们看看该函数流程.
	runtime.netpoll()
		-> epoll_wait()
			-> runtime.netpollready()
				-> netpollunblock(,,true) 激活阻塞的协程

6. netpollunblock()
	要介绍 netpollunblock() 包括后面 netpollblock() 的实现,我们需要重点分析一下 pollDesc.rg/wg 这两个字段.我们以 rg 举例.
	首先定义如下:
	type pollDesc struct {
		rg uintptr
	}

	该字段不仅用来表示阻塞的协程,还可以表示 i/o 是否 ready, 是否超时等.这些取值是互斥的,同时为了效率考虑,只使用了一个字段表示.
	我们看看此函数的具体实现:
	func netpollunblock(pd *pollDesc, mode int32, ioready bool) *g {
	gpp := &pd.rg
	if mode == 'w' {
		gpp = &pd.wg
	}

	// 状态迁移: for + cas
	for {
		old := atomic.Loaduintptr(gpp)
		1. 超时发生的时候,数据已经到达,则直接返回
		2. ioready 的时候, 直接返回
		if old == pdReady {
			return nil
		}

		1. 超时发生,如果没有协程阻塞,直接返回
		2. ioready 的时候, 即使 old == 0,需要设置 rg 为 pdReady
		if old == 0 && !ioready {
			// Only set pdReady for ioready. runtime_pollWait
			// will check for timeout/cancel before waiting.
			return nil
		}

		1. 超时发生,新状态设置为 0
		2. ioready 的时候,新状态设置为 pdReady
		var new uintptr
		if ioready {
			new = pdReady
		}

		1. 超时发生,将 rg 设置为 0, 并返回阻塞的 goroutine
		if atomic.Casuintptr(gpp, old, new) {
			if old == pdWait {
				old = 0
			}
			return (*g)(unsafe.Pointer(old))
		}
	}

7. netpollblock()

	几个小问题:
	1. 读写超时时间是不是会有偏差?
	2. go 定时器如何取消和修改呢?
		通过 go timer 机制的 modtimer 和 deltimer 函数. 那么 modtimer 在 timer heap 中是如何实现的呢?
		先 deltimer 再 addtimer,那么 deltimer 是如何实现的呢? 先使用 heap last 节点覆盖待删除节点,然后
		向上调整 || 向下调整(只需要调整一个方向即可).
	3. 阻塞在套接字读写上的 goroutine,重新调度时执行什么样的函数呢?毕竟重新调度有两个触发源,一个是数据到达,
	另一个是超时,执行的函数似乎不同.
	4. ...


问题:
1. netpoll_break 有什么作用?
	netpoll 在等待 socket ready 的时候,我们可以通过 netpoll_break 中断 netpoll 从而使得 netpoll 返回.
	那么 netpoll 是如何实现的呢?
	使用 pipe 创建一个非阻塞管道,netpoll 监听 rd, netpoll_break 只需要通过 wd 写入一字节数据即可.
2. epoll 各个 EP* 选项的区别?
	即使不注册 EPOLLHUB. EPOLLERR 事件,当事件发生时,我们仍然可以获取到.
	当对端关闭 socket 的时候,我们该如何检测此类事件呢?首先检查 EPOLLRD, 并且调用 read() 返回 0 即可.
	但这种方式涉及到一次系统调用,会影响性能.一种更加优雅的方式是使用 EPOLLRDHUP,只要该事件发生,即表示对方
	发送了 FIN 分节.
3. 注意 epoll_event 中 data 这种回调手法的使用
4. pollDesc 结构
5. 套接字的读写超时时间如何实现的呢?
	通过内部的定时器实现,读定时器的回调函数会检查是否有协程阻塞,如果没有,表示数据已经到达.如果有,表示数据还没有到达
	则把套接字从 netpoll 移除,并关闭 socket 发出错误.
*/
