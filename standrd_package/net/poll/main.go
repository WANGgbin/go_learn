package main

/*
探索 go 的网络输入输出模型
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
