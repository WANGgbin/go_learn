package main

import (
	"fmt"
	"time"
)

/*
 看一个统计函数时延场景下 defer 函数 + 闭包的巧妙使用
*/

func TimeCost() func(funcName string) {
	start := time.Now()
	return func(funcName string) {
		// 闭包函数捕获起始时间
		latency := time.Since(start).Milliseconds()
		fmt.Printf("func: %s, cost: %d ms", funcName, latency)
	}
}

func main() {
	// defer 会确定要执行的函数 和 函数的入参， 所以这里会执行 TimeCost()
	defer TimeCost()("main") // func: main, cost: 1004 ms%
	time.Sleep(1 * time.Second)
}
