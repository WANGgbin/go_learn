trace 是 go 自带的另一个程序 debug 工具.跟 pprof 搭配使用. trace 用来分析指定时间段内发生的所有事件.
与 pprof 一样, trace 的使用包括两步骤.
- 文件采集
  采集也包括三种方式.
  - http server
    path: `/debug/pprof/trace`
  - 插入 runtime 代码
    ```go
    import (
        "runtime/trace"
    )
    func main() {
        f, _ := os.Create("trace.out")
        defer f.Close()
        trace.Start(f)
        defer trace.Stop()
    }
    ```
  - go test -trace trace.out 
- 文件分析
  使用 `go tool trace trace.out` 来分析文件.同样有两种模式. 
  web 界面各个选项的含义可以参考:[golang性能分析指南:trace](https://www.modb.pro/db/231692). 我们重点关注两项:
  - `View trace`
    主要两栏.
    - 第一栏,包括协程数量变化, heap内存数量变化, thread 数量变化
    - 第二栏,包括 GC 事件(MARK, SWEEP 等), syscall 事件, proc 上发生的事件(哪个m哪个g什么样的函数).
  - `Goroutine analysis`
    每个协程的执行时间分析,总共用了多少时间,系统调用阻塞多少时间,网络阻塞多少时间,调度等待多少时间, GC 占用了多少时间.