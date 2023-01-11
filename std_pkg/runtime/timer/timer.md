# 问题
- go runtime 内部的 timer 使用的什么数据结构
- timer 是如何触发的? 是负责同步检测是否有可达的定时器?



# time 包中常见函数的底层实现
## timer.AfterFunc
函数功能是在 d 后,执行回调函数 f.

函数实现为:
```go
func AfterFunc(d Duration, f func()) *Timer {
	t := &Timer{
        // 初始化一个 timer
		r: runtimeTimer{
			when: when(d),
            // timer 并没有直接调用 f, 而是通过 goFunc 做了一层封装为什么?
			f:    goFunc,
			arg:  f,
		},
	}
	startTimer(&t.r)
	return t
}

// 我们看看 goFunc 的实现
func goFunc(arg interface{}, seq uintptr) {
    // 内部开启了一个新的协程来执行真正的函数
    go arg.(func())()
}
```
为什么不直接调用 f 而是开启一个协程来调用 f 呢? 实际上, runtime 是在 schedule() 等特定的地方执行到期的 timer 的. 对于用户指定的 f , 有可能需要执行很久,为了避免执行到期 timer 花费太长时间, 所以对于用户指定的或者运行时间比较长的 timer 会拉起一个新的 goroutine 去执行 f.

## timer.NewTimer
该函数会创建一个新的定时器,时间到达后, 会往 time.Timer 的 channel 中发送当前时间. 我们来看看函数的实现:
```go
type Timer struct {
    C <-chan Time
    // 特别注意, 这里是个值, 而不是指针. 在 After 场景下, 只有定时器 expire 后, Timer 空间才会被 gc
    r runtimeTimer
}

func NewTimer(d Duration) *Timer {
    // 管道大小为1
	c := make(chan Time, 1)
	t := &Timer{
		C: c,
		r: runtimeTimer{
			when: when(d),
            // sendTime 会将当前 Time 发送到 channel 中
			f:    sendTime,
			arg:  c,
		},
	}
	startTimer(&t.r)
	return t
}

func sendTime(c interface{}, seq uintptr) {
    // 非阻塞主要是用于 timer.NewTicker, 避免阻塞
	select {
	case c.(chan Time) <- Now():
    // 注意这种 select + default 实现非阻塞的方式
	default:
	}
}
```
## time.After
该函数是对 NewTimer 的简单封装,函数实现为:
```go
func After(d Duration) <-chan Time {
	return NewTimer(d).C
}
```
需要注意的是, 使用该函数, 底层的 Timer 只有定时器 expire 后, Timer 才能被垃圾回收.
## timer.Stop()
该函数用来停止一个定时器,函数实现为:
```go
// 如果 stop 了定时器则返回 true, 如果定时器已经到期或者已经停止,则返回 false.
func (t *Timer) Stop() bool {

	return stopTimer(&t.r)
}
```

## timer.NewTicker()
该函数用来创建一个周期性的定时器, 时间到达后,会往 chan Time 中发送当前时间,函数实现为:
```go
type Ticker struct {
	C <-chan Time // The channel on which the ticks are delivered.
	r runtimeTimer
}

func NewTicker(d Duration) *Ticker {
    // 如果 d <= 0, 则 panic
	if d <= 0 {
		panic(errors.New("non-positive interval for NewTicker"))
	}
	c := make(chan Time, 1)
	t := &Ticker{
		C: c,
		r: runtimeTimer{
			when:   when(d),
            // 周期时间设置为 d
			period: int64(d),
			f:      sendTime,
			arg:    c,
		},
	}
	startTimer(&t.r)
	return t
}
```

因为 Ticker 是周期性的, 如果我们不主动关闭的话, 这个定时器会一直运行. 因此当我们不需要 ticker 的时候,一定要调用 Stop() 方法来显示关闭次周期性定时器. 实际上 Timer 不会有这个问题,因为它是一次性的.

我们看看 Stop() 函数实现:
```go
func (t *Ticker) Stop() {
	stopTimer(&t.r)
}
```

我们还可以调用 Reset() 来重置 tikcer, 函数实现为:
```go
func (t *Ticker) Reset(d Duration) {
	if t.r.f == nil {
		panic("time: Reset called on uninitialized Ticker")
	}
	modTimer(&t.r, when(d), int64(d), t.r.f, t.r.arg, t.r.seq)
}
```

类似于 After, time 包提供了函数 Tick, Tick 本质上是对 NewTicker 的包装,函数实现为:
```go
func Tick(d Duration) <-chan Time {
	if d <= 0 {
		return nil
	}
	return NewTicker(d).C
}
```
**需要特别注意, Tick 只返回 chan, 底层的 Tick 被 p 的 timers 引用, 我们没有办法关闭此定时器,因此 Ticker 对应的空间是无法被 gc 的. 因此, Tick 函数只适用于那些并不关闭 ticker 的场景, 否则使用该函数会造成内存泄漏.**

# go timer 实现

通过分析 time 包中的函数, 我们发现, 函数实现依赖于 go runtime 的 startTimer, stopTimer, modTimer 函数. 我们来看看这些函数的底层实现.

go 中是通过 heap 来存放 timer 的. 为了性能考虑,每一个 p 维护一个 timer heap. timer 常见的操作有 add, delete, mod 等. 我们先来看看 timer 的定义:
```go
type timer struct {
	pp       uintptr  // 关联的 p
	when     int64  // 时间戳,单位: ns
	period   int64  // 周期性 timer 时间,单位: ns
	f        func(interface{}, uintptr) // NOTE: must not be closure 回调函数
	arg      interface{}  // 回调函数参数
	seq      uintptr
	nextwhen int64  // 用于 modtimer, modtiler 只是更新 timer 的 nextwhen. 会在 adjusttimers() 函数中,将 timer 调整到适当的位置
	status   uint32  // timer 状态, running, waiting, deleted, mod...
}
```
实际上, delete, mod 只是修改 timer 的状态. 在 特定的时间点, 会调用 `adjusttimers()` 函数来调整 p 的 timers, 包括真正删除 timer, 将 modified 的 timer 设置到正确的位置等. 然后会调用 `runtimer` 来执行到期的定时器. `adjusttimers()` 本质上就是对堆的调整.

关于 timer 的底层实现细节,可以参考 <深度探索Go语言> 6.10.