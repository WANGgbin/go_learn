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
  
    返回 400 错误，并关闭链接。
  
- 优雅关闭？
    
    - 首先关闭监听套接字
    - 关闭所有空闲的链接，如果存在非空闲的链接，通过二进制退避算法重试，直到关闭所有打开的链接。
  
- 怎么判断一条链接是否空闲
  
  为每一条链接维护了状态，当在一条链接上已经处理完一个 request 并等待处理下一个请求时，此时链接 的状态就是空闲的。参考：D:/install/go/src/net/http/server.go:2801

- server 端什么时候需要 关闭 一条链接
    
    - response.closeAfterReplay == true
      
        - 对于 http/1.0，如果 req 没有设置 Connection: keep-live。
        
        - req 设置 Connection: close.
        
        - server 层设置 keepAlivesEnabled == false
        
        - handler 注入 Connection: close
        
        - http1.0，如果无法确定 Content-Length，则通过 close connection 通知客户端 body 的结束。 http1.0 不支持 chunk 特性。
        
        - 在 chunkWriter.Write 的时候，在返回响应前，还会检测是否读取完 req.body，如果没有则尝试 consume req.body
        剩下的内容，如果遇到认为非 io.EOF 的错误，则设置 closeAfterReplay == true。因为，req 中剩余的内容不能作为写一个
        req 的一部分。
  
    - response.conn.werr != nil
      
        response 写的时候链接发生错误
      
    - response.contentLength != w.written
    
        response 写入的数据跟 contentLength 不相等时，关闭链接
  
- server 扩展点

  - handler
    
    server 有个 handler 成员，其类型是：
    ```go
          type Handler interface {
            ServerHttp(ResponseWriter, *Request)
          }
    ```
    
    默认的 handler 是 ServeMux 类型变量：DefaultServerMux，其内部定义了默认的路由规则。因此，

    如果我们要自定义路由规则，只需要实现 Handler 接口即可。实际上，go 世界很流行的 web 框架 Gin 就是通过这种方式来实现自定义的路由规则的。

    这一点给我们的启发是，在开发的时候，要留好扩展点，要遵循 DIP(Dependency Inverse Principle) 原则，上层依赖接口而不是具体实现， 这样可以尽量的降低彼此之间的耦合，方便维护和扩展。
    
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
  
    - 客户端错误
      返回 400
    - 业务错误
      通过 ResponseWriter.WriteHeader(statusCode) 指定业务指定错误码
    - 框架层错误
      关闭链接

- 在 httpHandler(w http.ResponseWriter, r *http.Request) 应该做什么

    - 注入必要的 Header
    - 指定状态码
    - 是否关闭 tcp 链接
    - 写入 body
    - 读取 request.body
    
        - 如果 body 长度 < request.Content-Length 会发生什么
          
            在调用 body.Read() 的时候，会返回 io.ErrUnexpectedEOF 错误。
          
        - 如果 handler 没有读取完毕 body 会发生什么
          
            实际上在发送响应前，会尝试读取完 body。或者 handler 可以通过 body.Close() 丢弃剩余的内容。
        - request.body.Close() 发生了什么
            
            丢弃 request body 剩余的内容。
    
- server 端写入流程

    - 包括 4 个 writer：
      
        response.w bufio.Writer ->
            response.cw chunkWriter ->
                response.conn.bufw bufio.Writer ->
                    checkConnErrorWriter ->
                        conn.rwc(底层的链接)
        
        可以看到上述流程封装了很多的 writer 为什么？
        - response.w -> response.cw 
          
            为了实现 http 的 chunk 特性，某些场景下，事先无法确定 body 的长度，此时就可以使用 chunk 特性。而这一层封装通过 bufsize 确定了 chunk 的大小，当缓冲区满的时候(除了最后一次 Flush)，便发送一次 chunk。
        
        - response.cw -> response.conn.bufw
          
            chunkWriter 除了实现 chunk 特性外，还有一个很重要的作用就是确定最终的 Header，比如：Content-Type、Content-Length、Connection、Transfer-Encoding 等。
        
        - conn.bufw -> checkConnErrorWriter
          
            有了缓冲区之后，只在必要的时候(缓冲区满、Flush)才会调用底层的 I/O，提高效率。
        
        - checkConnErrorWriter -> conn.rwc
        
            写错误的记录：response.werr，从而据此关闭底层链接
        
    - 另外一个特别需要学习的地方是，这一套写入流程应用了一种经典的设计模式：装饰器模式。
    
        都是 writer，通过装饰器模式，叠加一层层的功能，层层之间通过接口隔离实现。
    
- 几个重要的 Header
    看看几个重要的 Header 是如何确定的。
  
    - Content-Type
    
        如果在 handler 中没有指定的话， chunkWriter 会采用嗅探协议来确定 Content-Type.
      
    - Content-Length
        
        - handler 指定 Transfer-Encoding 头部，则不能指定 Content-Length 头部
        - handler 指定 Content-Length，正确性由 handler 保证。
        - handler 未指定 Content-Length 和 Transfer-Encoding
          
            - 如果 body 长度 > 2048，则采用 chunk 特性，指定 Transfer-Encoding: chunked， 不指定 Content-Length.
            - 否则，Content-Length = len(p)
    
    - Connection
    

