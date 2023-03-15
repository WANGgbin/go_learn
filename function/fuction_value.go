package main

import (
	"fmt"
	"strings"
)

/*

- funcval, 闭包对象, 闭包函数 的概念.
go 中函数被视为一种变量,可以作为函数的输入参数,也可以作为函数的返回值,同样也可以赋值给其他变量.
当函数赋值给一个变量的时候,我们就说这个变量是个 funcval. funcval 本质上是个指向某个 struct 的指针.
那么这个 struct 的定义是什么呢? 实际上,为了支持闭包, 该 struct 由 "函数指针 + 捕获列表" 组成.
该 struct 我们称之为 "闭包对象". 闭包对象中的函数指针指向的函数, 我们称之为 "闭包函数".
struct {
	F uintpr
	var1 type
	...
}

- 函数赋值给某个变量,底层发生了什么?
当我们把函数赋值给某个变量的时候,底层实际上会分配并初始化一个闭包对象, 变量的值就是这个闭包对象的值.

那如果是一些非闭包类型的函数, 将这些函数赋值给变量的时候,也需要每次分配并初始化闭包对象吗?
普通函数因为没有捕获列表,所以他的闭包对象实际上只包含一个函数指针,是固定不变的. 所以编译器在编译阶段就会全局初始化一个唯一的闭包对象.

- 捕获列表是如何传递给闭包函数的?
不同的闭包的结构体是不同的,所有闭包的类型元信息由编译器在编译阶段确定.
那么在调用闭包函数的时候,是如何访问捕获列表的呢?实际上, caller 在调用函数(不管是否是闭包函数,因为 caller 是无法区分某个函数是不是闭包函数)
前,会将只想函数对象的指针写入到某个寄存器(DX)中,在闭包函数内部,通过该寄存器来访问捕获列表中对应的参数.

- 闭包对象逃逸分析
因为 funcval 是个指向闭包对象的指针, 所以分配闭包对象的时候,也会进行逃逸分析. 闭包对象可能分配在堆上,也有可能分配在栈上.

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

//func main() {
//	n := 1
//	outter(n)()
//	fmt.Printf("%d\n", n)
//}

/*

注意在闭包函数内部定义与捕获列表中同名的变量引起的覆盖问题。
*/

func toUpper(s string) (int, string) {
	upper := strings.ToUpper(s)
	return len(s), upper
}

func main() {
	i := "main"
	f := func() int {
		var length int
		length, i = toUpper(i) // 左侧的 i 仍然是捕获列表中的 i
		fmt.Printf("%s\n", i)
		return length
	}

	fmt.Printf("before: %s\n", i)
	f()
	fmt.Printf("after: %s\n", i) // 输出：MAIN

	i = "main"
	f = func() int {
		length, i := toUpper(i) // 左侧的 i 是闭包函数内部的变量，并不是捕获列表中的 i，所以这里对 i 的修改并不会影响外部的 i
		fmt.Printf("%s\n", i)
		return length
	}

	fmt.Printf("before: %s\n", i)
	f()
	fmt.Printf("after: %s\n", i) // 输出：main
}
