描述 `net/http` 中客户端的实现。

# 问题
- 客户端连接池的管理
  
  - 内部通过一个 connectMethodKey -> idlCones 的 map 来维护
  空闲链接。
  
  - 通过 LRU 算法用来淘汰多余的空闲链接。


- 一条链接为什么要开两个协程，增加复杂度

  // Write the request concurrently with waiting for a response,
  // in case the server decides to reply before reading our full
  // request body.
  
  一个典型场景是，服务端还没完全读取 request，即判断有错误，然后返回一个 4xx 响应。
  
- resp.Body 为什么一定要调用 Close()

  如果不调用 Close()，读写的两个协程以及底层的 tcp 链接都无法关闭。为了能够实现链接的复用，
我们最好在读取完 resp.Body() 之后再调用 Close()。 因为如果不读取完 resp，那么在这条 tcp 
链接上，存在上一个 request 的脏数据，因此该 tcp 链接无法实现复用。
  
  可以参考：[resp.Body 为什么要 close](https://segmentfault.com/a/1190000042390597)
  
- http 包中 client 发起请求及收到响应的流程

  - 构造 request
    
  - 拉起两个协程
    
    协程是跟 tcp 绑定的，一个用于发送请求，另一个用于接受请求。
    
  - 通过 ch 把 request 发送给 write 协程， write 协程负责发送到 tcp
    
  - read 负责接受 response，并将 resp.Body 设置为 bodyEofSignal 类型，并将 response 通过 ch 发送给客户协程 \
  并等待应用层协程的信号，读完 resp / close
    
  - 应用层协程收到 resp 后，负责处理 resp. read/close. read/close 内部会通过 ch 发送信号给 read 协程
  
  - read 协程收到客户协程的信号后，决定复用链接还是关闭链接并释放两个读写协程
  

# 几种编程技巧

- 在编译期发现接口实现错误

  var _ io.ReaderFrom = (*persistConnWriter)(nil)

- switch 特殊用法

  switch 后无表达式，每个 case 的子表达式必须是 bool 类型。

```go
  switch {
	case t.Chunked:
		if noResponseBodyExpected(t.RequestMethod) || !bodyAllowedForStatus(t.StatusCode) {
			t.Body = NoBody
		} else {
			t.Body = &body{src: internal.NewChunkedReader(r), hdr: msg, r: r, closing: t.Close}
		}
	case realLength == 0:
		t.Body = NoBody
	case realLength > 0:
		t.Body = &body{src: io.LimitReader(r, realLength), closing: t.Close}
	default:
  }
```
