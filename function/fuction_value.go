package main

import "fmt"

/*
	go 中函数被视为一种变量,可以作为函数的输入参数,也可以作为函数的返回值,同样也可以赋值给其他变量.
实际上,为了支持闭包, go 中的函数并不是类似于 c 中的函数指针,而是一个指向函数对象的指针,该函数对象是一个由 "函数指针 + 捕获列表"
组成的结构体.

type funcval struct {
	F uintpr
	var1 type
	...
}

不同的闭包的结构体是不同的,所有闭包的类型元信息由编译器在编译阶段确定.
那么在调用闭包函数的时候,是如何访问捕获列表的呢?实际上, caller 在调用函数(不管是否是闭包函数,因为 caller 是无法区分某个函数是不是闭包函数)
前,会将只想函数对象的指针写入到某个寄存器(DX)中,在闭包函数内部,通过该寄存器来访问捕获列表中对应的参数.

*/

// n 传递仍然是值传递
func outter(n int) func() {
	// 在函数内部,会重新在 heap 上分配一个 int 变量初始化为 n 的值.
	// 之后,不管是在函数 outter 内部还是 闭包内部都是访问这个 heap 上的变量
	fmt.Printf("%d\n", n) // n 逃逸到 heap
	return func() {
		n++
		fmt.Printf("%d\n", n)
		return
	}
}

func main() {
	n := 1
	outter(n)()
	fmt.Printf("%d\n", n)
}
