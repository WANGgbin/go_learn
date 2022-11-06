package main

import (
	"fmt"
	"sync/atomic"

	"github.com/WANGgbin/golearn/plan9/my_atomic"
	// "github.com/WANGgbin/golearn/plan9/my_atomic"
)

// plan9 汇编参考:
// 1. https://mp.weixin.qq.com/s/eSVJSrC0IPzFBv3Nagmj6w (从 go 走进 plan9 汇编)
// 2.https://zhuanlan.zhihu.com/p/348227592 (肝了一上午的Golang之Plan9入门)

/*
	golang 使用的是 plan9 汇编, plan9 汇编可以认为是 Intel or AT&T 汇编的上层抽象,其目的在于
 方便我们编写汇编程序. 跟 Intel or AT&T 核心的区别在于 plan9 有 4 个逻辑寄存器.
	FP(Frame Pointer)
	使用该寄存器来访问函数的参数和返回值. eg: arg1+(FP)第一个参数,假设第一个参数 8 个字节, 则第二个参数为: arg2 +8(FP)
	SP(Stack Pointer)
	使用该寄存器来访问函数的局部变量. SP 有逻辑/物理两个寄存器.区分方法为:带 symbol 则为逻辑寄存去,否则为物理寄存器
	物理寄存器表示的是函数栈顶,那么逻辑寄存器表示的是什么呢? 是栈底, local1-8(SP) 表示第一个局部变量
	SB(Static Base Pointer) TEXT分节的起始地址
	PC(Program Counter) 跟物理寄存器 IP 一致

Stack frame layout(x86)

 +------------------+
 | locals of caller |
 +------------------+
 |   callee return2 |
 +------------------+
 |   callee return1 |
 +------------------+
 |   callee arg2    |
 +------------------+
 |   callee arg1    |
 +------------------+ <- logic FP
 |  return address  |
 +------------------+
 |  caller's BP (*) |
 +------------------+ <- logic SP
 |     local1       |
 +------------------+
 |     local2       |
 +------------------+
 |  args to callee  |
 +------------------+ <- real SP


 plan0 函数汇编声明
									局部变量 + 可能调用函数传参和返回值大小,不包括 ret address 大小
									  |		函数参数和返回值大小
									  |     |
TEXT ·pkgname.funcname(SB), NOSPLIT, $16 - 32

*/

var counter int64

func Incr(ch chan<- struct{}) {
	atomic.AddInt64(&counter, 1)
	ch <- struct{}{}
}

var old int64

func Swap(ch chan<- struct{}, val int64) {
	// atomic.SwapInt64(&old, val)
	my_atomic.SwapInt64(&old, val)
	ch <- struct{}{}
}

func main() {
	// ch := make(chan struct{})
	// for i := 0; i < 1000000; i++ {
	// 	go Incr(ch)
	// }
	tmp := int64(0)
	if my_atomic.CompareAndSwapInt64(&tmp, 0, 1) {
		fmt.Print("Swapped")
	}
}
