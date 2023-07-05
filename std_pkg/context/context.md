# 几个问题

context 相关内容可以参考：[深入理解 context](https://www.zhihu.com/tardis/zm/art/110085652?source_id=1005)

- 为什么需要 context

    我们经常会通过并发的方式来执行一个任务，如果发生异常、超时 或者 想主动取消这个任务，应该怎么处理呢？
我们可以使用 ch。
  
    这没有问题，但如果是任务中又嵌套任务呢？底层的协程需要同时监听多个 ch，这样才能收到所有的关闭信号。但是这样
  增加程序复杂度、降低可读性。
  
    为此 go 提出了 context 方案。
  
- context 是什么
  
    两个作用：
        
    - 通知下游取消任务
    - 携带 k/v
    

- 怎么使用
    
    - 创建 context
        - 创建空的 context
        
            - context.Background()
            - context.TODO()
        
        - 创建主动取消的 context
        
            ctx, cancelFunc := context.WithCancel(ctx)
        
        - 创建超时取消的 context
        
            - ctx, cancelFunc := context.WithTimeout(ctx, duration)
            - ctx, cancelFunc := context.WithDeadline(ctx, deadline)
    
            通过调用 cancelFunc() 主动关闭任务; 或者到期后，关闭任务。
        
        - 创建携带 k/v 的 context
        
            ctx := context.WithValue(ctx, key, value)
    
    - 获取 ctx 的到期时间
        
        ddl, ok := ctx.Deadline();
        
        如果没有设置到期时间，ok == false
      
    - 判断 ctx 是否被取消
        
        if err := ctx.Err(); err != nil {
            // 被取消，err 表示取消的错误原因，只有两种取值：context.Canceled: 主动取消; context.DeadlineExceeded: 超时
        } else {
            // 没有取消
        }
        
    - 监听 ctx 的取消信号
    
        <- ctx.Done(); 当 ctx 被取消时，关闭对应的 ch
      
    - 从 ctx 中提取 value
    
        value := ctx.Value(key)
    
- context 的实现

    - context.Context 定义
    ```go
      type Context interface {
        // 前三个函数跟取消有关
  
	    // 什么时候会取消
	    Deadline() (deadline time.Time, ok bool)
  
        // 监听取消信号
        Done() <-chan struct{}
        
        // 是否取消以及取消的原因是什么
        Err() error
  
        // 跟 value 有关
        Value(key interface{}) interface{}
    }  
    ```
  
    - context.Context 的实现
    
        总共 4 中实现
    
        - emptyCtx
    
            用于创建一个初始化的 ctx
    
        - valueCtx
    
            用于创建一个携带 k/v 的 ctx
    
        - cancelCtx
            
            ```go
            type cancelCtx struct {
                Context
            
                mu       sync.Mutex            // protects following fields
                done     chan struct{}         // created lazily, closed by first cancel call
                children map[canceler]struct{} // 上游就是通过这种方式给下游传递 cancel 信号的
                err      error                 // set to non-nil by the first cancel call
            }
            ```
        - timerCtx
    
            ```go
            type timerCtx struct {
                cancelCtx // timerCtx 是一种 cancelCtx
                timer *time.Timer // Under cancelCtx.mu.
            
                deadline time.Time
            }
    
            ```
          
- 使用例子

    在 http 包中，server 端在处理请求的时候，整个 server 会创建一个 ctx.
  
    当建立一条链接的时候，每一个 conn 又会创建一个 cancelCtx，当客户端关闭此 conn/或者 conn 发生异常的时候，关闭此 ctx.

    实际上，我们可以看到在 connReader 发生错误的时候，便通过 handleReadError() 关闭了链接维度的 ctx. 在 checkConnErrorWriter 
写错误的时候也会关闭此 ctx. 或者如果设置 closeAfterReplay()，也会关闭此 ctx.
  
    当从链接获取到一个请求的时候，又会创建一个 cancelCtx. 当处理 req 失败/完毕的时候，就关闭此 ctx.
  
    实际上，我们并没有看到什么地方监听了这些 ctx。那为什么还要创建 ctx 呢？
  
    这是一种抽象，也是一种扩展性。 conn 本身就代表了一个任务，在某些特殊场景下，需要关闭此任务，那么我们就应该创建一个
  对应的 ctx。 req 同理。