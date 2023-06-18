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
  