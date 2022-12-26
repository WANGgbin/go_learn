# pprof
pprof 是 go 自带的程序 debug 工具. 如何使用 pprof 调试 go 程序呢?<br>
整个调试工程分为两步.
- 获取采样文件
- 分析采样文件

pprof 可以用来分析程序的哪些特性呢?
- heap
  函数内存分配情况,有四种类型.
  - inuse_objects 已分配未释放对象个数
  - inuse_space  已分配未释放内存空间大小
  - alloc_objects 总共分配对象个数(包括已释放对象)
  - alloc_space 总共分配空间大小(包括已释放对象)

  默认类型为 inuse_space,可以直接在交互模式中输入类型的名称来更换类型.在 web 界面的 `SAMPLE` tab 中也可以更换类型.
- cpu
  函数 cpu 使用时间分析
- goroutine
  协程函数调用栈分析

## 获取采样文件
通常有三种方式来生成采样文件.
- http 服务器
- 程序中插入 runtime 代码
- go test 通过命令行参数指定

这里以获取堆内存采样文件举例.
- http 服务器
  - 首先需要在 go 程序中启动 http 服务器
    ```go
    import (
        _ "net/http/pprof" // 注册 http 路由
    )

    func main() {
        // 启动 http 服务器
        go func(){
            http.ListenAndServe(":6060", nil)
        }()

        // do something
    }
    ```
  - 获取采样文件
    ```sh
    curl -o heap.out http://localhost:6060/debug/pprof/heap
    # or
    go tool pprof http://localhost:6060/debug/pprof/heap # 获取采样文件并同时打开交互模式分析采样文件
    ```
    其他类型文件的 path 可以参考 "net/http/pprof" 包.
    - heap: /debug/pprof/heap
    - goroutine: /debug/pprof/goroutine # 协程调用栈分析
    - cpu: /debug/pprof/profile?seconds=20  # 函数执行时间分析
    - threadcreate: /debug/pprof/threadcreate # 创建新线程的堆栈

- 程序中插入 runtime 代码**TODO**
- go test
```sh
go test -memprofile=mem.out ... # 内存分配信息
go test -cpuprofile=cpu.out ... # cpu 使用信息
go test -blockprofile=blocl.out ... # 阻塞信息
go test -mutexporfile=mutex.out ... # 互斥数据
go test -trace=trace.out # trace 信息
```
## 分析采样文件
我们使用 go 自带的 pprof 工具来分析收集到的采样文件.
整体来说, pprof 以函数调用链为主干来分析各项数据. 比如在 heap 分析中, 函数方块表示此函数分配内存情况.箭头表示多少内存分配是由子函数触发的. 再比如在 goroutine 分析中, 函数方块表示有多少协程调用了此函数.

pprof 有两种模式:
- 交互模式
  交互模式下,最常用的有两个命令.
  - top n
    展示 top n 函数
  - list func_name
    展示此函数详细信息,比如分配内存很多,到底是由函数中哪条语句导致的,就可以通过 list 函数查看.
  - web
    web ui 中展示信息
- 可视化模式
默认是交互模式, 要打开可视化模式.需要以下两步:
- 安装 `graphviz`
```sh
    sudo apt-get install graphviz
```
- 开启 http 服务器
```sh
    go tool pprof -http ":8080" file
```
可视化模式以一种可读性更好的方式展示结果,在 `VIEW` tab 中各个选项的含义为:
- Top
    top 信息,对应交互模式中的 top 命令
- Grap欻
    对应交互模式中的 web 命令
- Flame Graph
    传说中的火焰图,调用链路为从上往下.
- Source
    对应交互模式中的 list 命令
- Disassemble
    总计,比如分配次数,总耗时,总协程数等等.