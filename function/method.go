package main

import (
	"fmt"
)

type person struct {
	name string
}

func (p *person) getName() string {
	return p.name
}

/*

方法的调用本质上和函数的调用是一样的,只不过会将接受者作为第一个参数.
方法的接受者有两种类型:值接受者 和 指针接受者.

1. 方法值
当我们将一个与对象绑定的方法赋值给某个变量的时候,这个变量就是方法值. 方法值本质上和函数值一样也是个指向闭包对象的指针. 当使用方法值的时候,
编译器会生成一个包装原始方法函数的闭包函数.
*/

// func main() {
// 	var mVal func() string
// 	p := &person{name: "wgb"}
// 	// mVal 为方法值
// 	mVal = p.getName

// 	fmt.Print(mVal())  // output: wgb
// }

/*

2. 方法表达式
我们可以像调用普通函数一样直接调用方法. 注意对于指针接受器和值接收器,调用方法是不相同发的.
指针接收器:
(*type)method()
值接收器:
(type)method()

因为 method 是相对于某个特定的类型而言的.

*/

// type person struct {
// 	name string
// }

// func (p person) getName() string {
// 	return p.name
// }

// func (p *person) setName(name string) {
// 	p.name = name
// }

// func main() {
// 	fmt.Print(person.getName(person{name: "wgb"})) // 输出: wgb
// 	tmp := &person{}
// 	(*person).setName(tmp, "wgb")
// }

/*

3. 为什么要为值类型的方法生成对应的指针类型的方法?

一方面,可以基于值类型方法生成基于指针类型的方法.

更重要的是, 当以一个接口来调用值类型的方法的时候, 编译器需要知道对象的大小,这样才可以拷贝一个完整的对象传递调用值对象方法.
实际上在编译期, 编译器很难知道接口指向值类型对象的大小. 一种方法是在运行期根据值类型的类型元信息来获取类型的大小,但是这样
性能比较低.

因为很容易获取对象的指针, go 编译器通过为每一个值类型方法生成一个指针方法, 从而在调用值类型方法的时候, 只需要通过指针调用相应的函数即可.

此外, go 语言还提供了语法糖, 使得我们可以通过指针/值 来调用 值/指针对应的方法.

*/

type Object interface {
	getName() string
	setName(name string)
}

type student struct {
	name string
}

func (s student) getName() string {
	return s.name
}

func (s *student) setName(name string) {
	s.name = name
}

func main() {
	var o Object
	o = &student{}

	// 指针类型虽然没有定义 getName() 方法, 但是基于值类型的 getName() 会生成一个方法
	o.setName("wgb")
	fmt.Print(o.getName()) // output: wgb

	// 通过指针调用值类型的方法. 原理: 先通过指针拷贝得到一个值对象,然后调用值对象对应的方法.
	p := &student{name: "wgb"}
	fmt.Print(p.getName())

	// 通过值调用指针类型的方法. 原理: 直接取值对象的指针,然后调用指针对象对应的方法.
	v := student{}
	v.setName("wgb")
	fmt.Print(p.getName())

}
