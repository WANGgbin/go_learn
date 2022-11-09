package main

/*
探索 go 的网络输入输出模型
go 的网络模型位于内核和应用层之间,既要保证应用层的易用,又要保证高性能.接下来从这两个角度来分析下 go
网络标准库的实现.

1.应用层的易用性
	要给用户提供易用的接口,就需要精心设计,封装.
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
	比如 gc,监控县城,调度器都会调用 netpool 函数,只有监控线程会设置 > 0的值,其他调用都设置为0.



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
