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
