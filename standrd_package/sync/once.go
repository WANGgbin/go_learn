package main

/**
	sync.Once 让某个行为只执行一次,那么是如何实现的呢?

猜测:
	类似于单例模式,使用一个标记,判断是否执行过函数.

go 实现:
	跟我们猜测的一致,实现比较简单.具体可以参考 go 源代码.
*/

import (
	"fmt"
	"sync"
)

func main() {
	var once sync.Once

	f := func() {
		fmt.Printf("do something\n")
	}

	done := make(chan struct{})

	for i := 1; i <= 10; i++ {
		go func() {
			once.Do(f)
			done <- struct{}{}
		}()
	}

	for i := 1; i <= 10; i++ {
		<-done
	}
}
