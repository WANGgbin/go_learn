描述 `net/http` 包中服务端的实现。

# 问题

- 跟 tcp 层是如何交互的
  
    并没有直接调用 tcp 相关的 api，而是调用通用的 api，维护的是 net.Listener 的接口。达到
弱耦合的目的。

- 路由注册
  
    http 包内部维护了 uri pattern 到 handler 的 map. 通过该 map 来查找对应的 handler.

- 并发处理模型，是否对协程数量进行限制
  
  通过 Accept() 获取一条链接，然后由一个协程负责处理一个链接，标准包**没有对协程的数量进行限制**

- 读写超时时间是如何实现的
  
    使用 runtime 自定义的定时器实现

- 如何使用 context 的
  

- 协程错误处理
  
    每个协程都会 recover，防止进程的 panic.

- tls 层是如何实现的
- 通过什么样的方法来保证高性能
  
    - 通过 sync.Pool 的方式来维护一条链接常用的数据结构，从而提高效率。 
    - 通过 bufio 实现提高 io 性能, **注意 bufio 这种装饰器模式的应用，在 src 之上增加功能的时候，就应该考虑装饰器模式**

- 如果请求的长度 > Content-Length 指定的长度，怎么处理
- 优雅关闭？
    
    - 首先关闭监听套接字
    - 关闭所有空闲的链接，如果存在非空闲的链接，通过二进制退避算法重试，直到关闭所有打开的链接。
  
- 怎么判断一条链接是否空闲
  
  为每一条链接维护了状态，当在一条链接上已经处理完一个 request 并等待处理下一个请求时，此时链接
  的状态就是空闲的。参考：D:/install/go/src/net/http/server.go:2801

- server 端什么时候需要 reuse 一条链接

    - http/1.0 Connection: Keep-Live
    - http/1.1 默认重用链接，如果指定 Connection: close 则关闭链接
  
- server 扩展点

  - handler
    
    server 有个 handler 成员，其类型是：
    ```go
          type Handler interface {
            ServerHttp(ResponseWriter, *Request)
          }
    ```
    
    默认的 handler 是 ServeMux 类型变量：DefaultServerMux，其内部定义了默认的路由规则。因此，
如果我们要自定义路由规则，只需要实现 Handler 接口即可。实际上，go 世界很流行的 web 框架 Gin
就是通过这种方式来实现自定义的路由规则的。

    这一点给我们的启发是，在开发的时候，要留好扩展点，要遵循 DIP(Dependency Inverse Principle) 原则，上层依赖接口而不是具体实现，
这样可以尽量的降低彼此之间的耦合，方便维护和扩展。
    
- server 侧的 header 是如何注入的？

  ResponseWriter 有一个 Header 方法，用来返回当前的 Header，我们可以通过调用 Header 相关的方法来设置 Header。
  eg:
  ```go
    func(w http.ResponseWriter, req *http.Request) {
        w.Header().Set("content-type", "application/json")
        w.Header().Add("some-key", "extra-value")
    } 
  ```
  
  会到 http.response， Header() 实际上返回的是 response.handlerHeader.
  
  除了业务层主入 Header 外，框架本身还会注入一些额外的 Header。而这是在 chunkWriter 负责写入的。
  关于 http 的 chunk 模式可以参考：[http协议: 大文件传输](https://zhuanlan.zhihu.com/p/390935751)
  
  - chunkWriter
    - WriteHeader
      负责注入额外的 Header，主要包括：
        - Content-Length
        - Content-Type: 如果应用层没有注入，采用嗅探算法设置 body type
        - Transfer-Encoding：http1.1 及以上，如果没设置 Content-Length，则设置为 Chunk，开启 Chunk 模式
        - Connection：关闭链接还是打开链接
    - Write
      ```go
    func (cw *chunkWriter) Write(p []byte) (n int, err error) {
	if !cw.wroteHeader {
		cw.writeHeader(p)
	}
	if cw.res.req.Method == "HEAD" {
		// Eat writes.
		return len(p), nil
	}
    // chunk 协议就是 length\r\ncontent\r\n
    // response.bufw = bioWriter(chunkWriter, 2048)
    // reponse.bufw 设置为 bio 的目的就是为了设置 chunk 的大小。
	if cw.chunking {
		_, err = fmt.Fprintf(cw.res.conn.bufw, "%x\r\n", len(p))
		if err != nil {
			cw.res.conn.rwc.Close()
			return
		}
	}
	n, err = cw.res.conn.bufw.Write(p)
	if cw.chunking && err == nil {
		_, err = cw.res.conn.bufw.Write(crlf)
	}
	if err != nil {
		cw.res.conn.rwc.Close()
	}
	return
}
      ```

- server 发生错误的时候如何响应？业务层错误？框架层错误？


