package main

import (
	"fmt"
	"time"
)

/*
go timer 的实现
1. go 维护 timer 的结构是红黑树吗?
2. go 是如何定时查看是否有 timer 的呢? go 的 ticker 是如何实现的呢?我们知道 os 的 ticker
是通过振荡器发出中断实现的.
3. go 定时器中的回调函数是如何实现的呢?
4. go 的 timer 机制中,是串行执行到期的 timer 函数的,因此一定要注意 timer 中设定的函数要足够高的性能
以及不能阻塞. 那么我们通过 timer 包设置定时器的时候, 是不是也应该如此呢?实际上不会, timer 包会基于用户
提供的函数在封装一个函数,而该函数是通过 goroutine 来执行用户提供的函数的,所以真正添加到 go timer heap
中的函数仍然可以很快就返回.

比如: time.AfterFunc(),真正注册的定时器函数是 goFunc.

func AfterFunc(d Duration, f func()) *Timer {
	t := &Timer{
		r: runtimeTimer{
			when: when(d),
			f:    goFunc,
			arg:  f,
		},
	}
	startTimer(&t.r)
	return t
}

func goFunc(arg interface{}, seq uintptr) {
	go arg.(func())()
}

*/

func main() {
	ch := make(chan struct{})
	time.AfterFunc(3*time.Second, func() {
		fmt.Printf("send notification\n")
		close(ch)
	})
	<-ch
	fmt.Printf("recieve notification\n")
}
