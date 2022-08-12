package main

import (
)
// go 中常量与变量之间的隐式转化
// go 中常量可以分为：
// 按命名分：
//	1、未命名常量 eg: 1/2/3.4
//	2、命名常量   eg: const PI = 3.14
// 按类型分：
//	1、有类型常量 eg: const PI float64 = 3.14
//	2、无类型常量 eg: const PI = 3.14 、1

// go规定，两个无类型常量进行运算的时候，优先级为 整型 < rune < 浮点型 < 复数
const A = 3.14
const B = A * 1
const C = 3.14 * 1 // A、B、C 均为浮点型

const I int = 1
const J = I * 3.0 // 由于 I 是整型，所以 J 也是整型

// func main() {
// 	fmt.Printf("%T %T %T %T %T\n", A, B, C, I, J) // output: float64 float64 float64 int int
// }

// 常量与变量转化
// 只需要记住一点：不能溢出/截取，才可以隐式转化，除非强转/显式转化

func main() {
	// var i int = C  // 这里 C 是浮点数，转化为 int 会造成精度损失，所以会报错
	// var j int = 3.0 // 3.0 虽然是浮点数，但是不会造成精度损失，所以不会报错
	// const K = I * 3.1 // 这条规则同样适用于常量之间的运算，这里 3.1 会转化为整型，但是有精度损失，报错
}
