描述 go 如何封装 tcp 客户端套接字。

# 几个问题

- 非阻塞套接字？
    
    是的，整个 go I/O 模型都是非阻塞的。在创建套接字的时候就设置为非阻塞了。
  
- 默认设置了哪些套接字选项？
    
    - SOCK_NONBLOCK | SOCK_CLOEXEC
      
        在创建套接字的时候就创建。不仅仅是客户端套接字，其他类型套接字都是如此。参考：
      
        ```go
            func sysSocket(family, sotype, proto int) (int, error) {
                s, err := socketFunc(family, sotype|syscall.SOCK_NONBLOCK|syscall.SOCK_CLOEXEC, proto)
                // On Linux the SOCK_NONBLOCK and SOCK_CLOEXEC flags were
                // introduced in 2.6.27 kernel and on FreeBSD both flags were
                // introduced in 10 kernel. If we get an EINVAL error on Linux
                // or EPROTONOSUPPORT error on FreeBSD, fall back to using
                // socket without them.
                switch err {
                case nil:
                    return s, nil
                default:
                    return -1, os.NewSyscallError("socket", err)
                case syscall.EPROTONOSUPPORT, syscall.EINVAL:
                }
            
                // See ../syscall/exec_unix.go for description of ForkLock.
                syscall.ForkLock.RLock()
                s, err = socketFunc(family, sotype, proto)
                if err == nil {
                    syscall.CloseOnExec(s)
                }
                syscall.ForkLock.RUnlock()
                if err != nil {
                    return -1, os.NewSyscallError("socket", err)
                }
                if err = syscall.SetNonblock(s, true); err != nil {
                    poll.CloseFunc(s)
                    return -1, os.NewSyscallError("setnonblock", err)
                }
                return s, nil
            }
        ```     
      
    - syscall.TCP_NODELAY
      
        该选项表示有数据后立刻发送。
      
    - syscall.SO_KEEPALIVE
      
        如果 dialer.KeepAlive >= 0，则会打开套接字 syscall.SO_KEEPALIVE 选项，并设置 syscall.TCP_KEEPINTVL | syscall.TCP_KEEPIDLE 表示
      多久后发送保活探测报文。
      
    - syscall.SO_REUSEADDR
        
        如果是服务端套接字还会设置该选项，可以重用端口。

- 如何设置自定义的套接字选项？
  
    对于 tcp 套接字，可以调用 TCPConn. Set* 类方法来设置套接字选项，详情参考：net/tcpsock.go 文件。 可设置的选项主要包括：
    
    - syscall.SO_LINGER
    - syscall.SO_KEEPALIVE
    - syscall.TCP_KEEPINTVL | syscall.TCP_KEEPIDLE
    - syscall.TCP_NODELAY
  
- 非阻塞 connect 套接字，如何判断连接成功还是发生错误？
    
    非阻塞套接字发起 connect 调用后，如果没连接成功，则会返回 `EINGRESS` 错误。 当连接成功后，套接字变的可写，因为目前套接字写缓冲区是空的而且
  还没有发送过数据。 当套接字发生错误的时候，套接字变的可读可写。
  
    那么怎么区分这两种情况呢？**通过套接字错误选项 SO_ERROR 区分**， 参考下面代码：
    ```go
         for {
		// Performing multiple connect system calls on a
		// non-blocking socket under Unix variants does not
		// necessarily result in earlier errors being
		// returned. Instead, once runtime-integrated network
		// poller tells us that the socket is ready, get the
		// SO_ERROR socket option to see if the connection
		// succeeded or failed. See issue 7474 for further
		// details.
        
        // 当注册套接字之后，协程调用 WaitWrite() 阻塞，直到连接成功/失败 
		if err := fd.pfd.WaitWrite(); err != nil {
			select {
			case <-ctx.Done():
				return nil, mapErr(ctx.Err())
			default:
			}
			return nil, err
		}
  
        // 获取套接字 SO_ERROR 选项值
		nerr, err := getsockoptIntFunc(fd.pfd.Sysfd, syscall.SOL_SOCKET, syscall.SO_ERROR)
		if err != nil {
			return nil, os.NewSyscallError("getsockopt", err)
		}
		switch err := syscall.Errno(nerr); err {
        
        // 此三类错误，重试
		case syscall.EINPROGRESS, syscall.EALREADY, syscall.EINTR:
		case syscall.EISCONN:
			return nil, nil
		case syscall.Errno(0):
            // 连接成功返回
			// The runtime poller can wake us up spuriously;
			// see issues 14548 and 19289. Check that we are
			// really connected; if not, wait again.
			if rsa, err := syscall.Getpeername(fd.pfd.Sysfd); err == nil {
				return rsa, nil
			}
		default:
            // 其他情况返回错误
			return nil, os.NewSyscallError("connect", err)
		}
		runtime.KeepAlive(fd)
	}   
    ```
  
- 注入到 netpoll 中的套接字默认的 epoll 选项是什么？

    无论是客户端套接字还是服务端套接字还是普通的文件描述符，注册到 netpoll 中的默认选项是：`EPOLL_IN | EPOLL_OUT | EPOLL_READHUP | EPOLL_ET`
    ```go
        func netpollopen(fd uintptr, pd *pollDesc) int32 {
            var ev epollevent
            ev.events = _EPOLLIN | _EPOLLOUT | _EPOLLRDHUP | _EPOLLET
            *(**pollDesc)(unsafe.Pointer(&ev.data)) = pd
            return -epollctl(epfd, _EPOLL_CTL_ADD, int32(fd), &ev)
        }
    ```

- connect() 的超时如何实现

    在创建套接字发起连接并注册到 go netpoll 中后，会根据 ctx.Deadline() 判断是否设置超时时间以及超时时间是什么。然后通过
  `poll.FD.SetWriteDeadline()` 设置超时时间，并在连接成功之后清理相关定时器。代码如下：
  
    ```go
        if deadline, hasDeadline := ctx.Deadline(); hasDeadline {
            fd.pfd.SetWriteDeadline(deadline)
            defer fd.pfd.SetWriteDeadline(noDeadline)
        }
    ```
  
    之后，当前协程便通过 `poll.FD.WaitWrite()` 等待连接成功或者出错。但是这样就可以了吗？
  
    如果上层调用者 cancel 了 ctx，等待套接字连接成功的协程如何感知到该信号并退出呢？ net 标准包给出的方案是，启动一个异步监控
  协程，当监控到 ctx 被 cancel，通过 `poll.FD.SetWriteDeadline()` 设置一个过去的时间，从而立刻激活阻塞协程。相关代码如下：
  
    ```go
        // Start the "interrupter" goroutine, if this context might be canceled.
        // (The background context cannot)
        //
        // The interrupter goroutine waits for the context to be done and
        // interrupts the dial (by altering the fd's write deadline, which
        // wakes up waitWrite).
      
        if ctx != context.Background() {
		// Wait for the interrupter goroutine to exit before returning
		// from connect.
		done := make(chan struct{})
		interruptRes := make(chan error)
		defer func() {
            // 通知监控协程退出
			close(done)
            // 确保监控协程退出
			if ctxErr := <-interruptRes; ctxErr != nil && ret == nil {
				// The interrupter goroutine called SetWriteDeadline,
				// but the connect code below had returned from
				// waitWrite already and did a successful connect (ret
				// == nil). Because we've now poisoned the connection
				// by making it unwritable, don't return a successful
				// dial. This was issue 16523.
				ret = mapErr(ctxErr)
				fd.Close() // prevent a leak
			}
		}()
        
        // 启动异步监控协程
		go func() {
			select {
			case <-ctx.Done():
				// Force the runtime's poller to immediately give up
				// waiting for writability, unblocking waitWrite
				// below.
                
                // aLongTimeAgo 是一个过去的时间，对于 SetDeadline() 函数而言，当设置为一个过去时间，如果已经有相关定时器，
                // 删除已有的定时器，同时立刻激活任何阻塞在 i/o 上的协程。 详情参考：runtime/netpoll.go: poll_runtime_pollSetDeadline
				fd.pfd.SetWriteDeadline(aLongTimeAgo)
                // 告诉主协程本协程已完成
				interruptRes <- ctx.Err()
			
            // 这个分支很重要，如果上层没有取消 ctx 且 连接成功，则通过监控 done 来退出监控协程，否则会造成协程泄露。
            case <-done:
				interruptRes <- nil
			}
		}()
	}
    
    for {
		if err := fd.pfd.WaitWrite(); err != nil {
            // 这里为什么要 case <-ctx.Done() 呢？
            // 因为从 WaitWrite() 返回，很有可能是因为上层取消 ctx 导致的，因此在这里需要判断这种情况，
            // 如果是则返回 ctx.Err()。
			select {
			case <-ctx.Done():
				return nil, mapErr(ctx.Err())
			default:
			}
			return nil, err
		}
		// ...
	}  
    ```
  
- context 的使用
    
    什么时候才应该监控 ctx.Done() 呢？
  
    - 如果操作分成几个重要的 step，可以在每个 step 前判断一次上下文是否结束。
    - TODO(@wangguobin)