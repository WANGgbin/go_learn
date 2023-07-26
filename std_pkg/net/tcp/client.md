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
- 