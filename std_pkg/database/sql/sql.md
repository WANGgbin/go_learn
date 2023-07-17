学习如何跟数据库进行交互。

# 几个问题
- ping() 如何实现的？
- mysql 协议？

mysql 协议格式还是比较简单的，如下所示：

| 3 Byte Length | 1 Byte Sequence |<br>
|           Payload               |

- mysql 跟 golang 数据类型对比

- database/sql 使用
  
  参考：https://juejin.cn/post/7127580456001732645
  
    - CRUD
    - 事务
      
    - sql 注入
      
    本质上，就是后端没有对前端的输入进行校验，直接通过前端的输入拼接 sql，进而导致一些预期之外的数据库操作。
  
    - 预处理
      
      https://blog.csdn.net/Stannis/article/details/120331281
      
      - 预处理的优势
        - 高性能，不用每次都分析语句。
        - 节省带宽，每次传递更小的数据
        
      - 定义预处理语句
        ```sql
        PREPARE stmt_name FROM 'sql';
        ```
      - 执行预处理语句
        ```sql
        EXECUTE stmt_name USING @var_name, @var_name; -- 必须通过变量的方式传参
        ```
      - 删除预处理语句
        ```sql
        (DEALLOCATE | DROP) PREPARE stmt_name; -- 预处理是占用服务器资源的，所以不适用的时候应该尽早释放
        ```

- sql 是如何扩展的

  我们知道有多种类型的数据库，那么 go 的标准库实现了哪一种数据库呢？

  实际上，这是一个接口的典型应用场景，类似于模板模式，标准库负责定义各种接口以及数据库的模板操作，由具体的 sql drive
来实现这些接口。
  
  - 分层模型
  
  - 哪些接口
  
  - 连接池
  
    设计一个连接池，我们需要考虑什么呢？
  
    - 客户端打开最大连接数量
    
      为什么要设置这个值呢？这实际上是对服务端的保护。在高并发的场景下，如果不对最大打开连接数进行限制，很可能导致服务端打开很多的连接，进而导致
      oom，进而导致服务进程挂掉。
      
      那如果已经达到最大打开连接数，新的连接如何处理呢？有两种方式：报错 或者阻塞等待。
      
      database/sql 给出的方案是阻塞等待直到有空闲的连接，当然可以通过 ctx 设置最大等待时间。
      
    - 连接池空闲连接的最大数量
      
      避免资源浪费
      
    - 连接最大空闲时间
  
      如果一个连接在最大空闲时间这个跨度内都没有被使用，显然当前客户端的请求是不频繁的，应该释放掉不适用的资源，避免资源浪费。
  

  - 具体数据库的 driver 如何接入 go 标准库
  
  - tx 实现
    
    DB.BeginTx(ctx context.Context) 支持当 ctx 被取消的时候，事务回滚。我们来看看实现细节。
  
    为了能够及时监控到 ctx 被取消的信号，在创建 tx 的时候，开启了一个协程负责监听该信号。当监听到 ctx 被取消的时候，rollback。
  
    但是，当 ctx 被取消的时候，我们实际上并不知道 tx 执行到什么阶段了。如果已经 commit/rollback 了，那么监控协程就不应该再次 rollback。
  
    或者 当监控协程 rollback 后，业务协程也不应该再次 commit/rollback。这一步怎么实现呢？
  
    通过一个标记字段，在每次 commit/rollback 的时候，原子性设置(CompareAndSwap)，如果设置成功，则表示还没 commit/rollback，则进行操作，
    反之，则表明已经有协程负责 commit/rollback 了，放弃操作。
    
    这里的标记字段就是 `tx.done` 字段。有了标记字段，便解决了多次 commit/rollback 的问题。
  
    但是为什么要创建一个 tx 专属的 ctx 呢？为什么不直接使用已有的 ctx 呢？
  
    考虑这样一个问题，假设事务进行了正常的提交，显然还需要通知监控协程退出。显然，需要关闭 tx 相关的 ctx，不应该是 scope 更广的 ctx。这就是
    为什么要新建一个 ctx。这是一个 ctx 使用的经典例子，值得反复学习。
    
    我们来分析下 database/sql 中关于 tx 的实现。
  
    - tx.Begin()
```go
func (db *DB) Begin() (*Tx, error) {
	return db.BeginTx(context.Background(), nil)
}

func (db *DB) begin(ctx context.Context, opts *TxOptions, strategy connReuseStrategy) (tx *Tx, err error) {
	// 从连接池获取一个空闲的连接
	dc, err := db.conn(ctx, strategy)
	if err != nil {
		return nil, err
	}
	// 创建事务
	return db.beginDC(ctx, dc, dc.releaseConn, opts)
}

// beginDC starts a transaction. The provided dc must be valid and ready to use.
func (db *DB) beginDC(ctx context.Context, dc *driverConn, release func(error), opts *TxOptions) (tx *Tx, err error) {
	var txi driver.Tx
	keepConnOnRollback := false
	withLock(dc, func() {
		// 通过接口 driver.Conn 创建一个事务接口 driver.Tx
		txi, err = ctxDriverBegin(ctx, opts, dc.ci)
	})
	if err != nil {
		release(err)
		return nil, err
	}

	// Schedule the transaction to rollback when the context is cancelled.
	// The cancel function in Tx will be called after done is set to true.
	// 创建一个可取消的子 ctx，在事务提交/回滚的时候，cancel ctx，通知监控协程退出。
	// 同时，如果父 ctx 被取消了，监控协程负责回滚事务。
	ctx, cancel := context.WithCancel(ctx)
	tx = &Tx{
		db:                 db,
		dc:                 dc,
		releaseConn:        release,
		txi:                txi,
		cancel:             cancel,
		keepConnOnRollback: keepConnOnRollback,
		ctx:                ctx,
	}
	
	// 开启一个协程，监控 ctx.Done()，当父 ctx 被取消的时候，回滚事务。
	go tx.awaitDone()
	return tx, nil
}
```
  
    - tx.Commit()
```go
func (tx *Tx) Commit() error {
	// 首先判断是否已经回滚，监控协程可能已经回滚事务。
	if !atomic.CompareAndSwapInt32(&tx.done, 0, 1) {
		return ErrTxDone
	}

	// cancel tx.ctx，使得监控协程退出。
	tx.cancel()
	tx.closemu.Lock()
	tx.closemu.Unlock()

	var err error
	withLock(tx.dc, func() {
		// 调用 driver.Tx.Commit() 完成事务提交
		err = tx.txi.Commit()
	})
	if err != driver.ErrBadConn {
		// 关闭事务所有的预处理语句
		tx.closePrepared()
	}
	// 将连接扔到线程池
	tx.close(err)
	return err
}
```
    - tx.Rollback()
```go
// Rollback() 跟 Commit() 基本一致，不再赘述。
func (tx *Tx) rollback(discardConn bool) error {
	if !atomic.CompareAndSwapInt32(&tx.done, 0, 1) {
		return ErrTxDone
	}

	if rollbackHook != nil {
		rollbackHook()
	}

	// Cancel the Tx to release any active R-closemu locks.
	// This is safe to do because tx.done has already transitioned
	// from 0 to 1. Hold the W-closemu lock prior to rollback
	// to ensure no other connection has an active query.
	tx.cancel()
	tx.closemu.Lock()
	tx.closemu.Unlock()

	var err error
	withLock(tx.dc, func() {
		err = tx.txi.Rollback()
	})
	if err != driver.ErrBadConn {
		tx.closePrepared()
	}
	if discardConn {
		err = driver.ErrBadConn
	}
	tx.close(err)
	return err
}
```

