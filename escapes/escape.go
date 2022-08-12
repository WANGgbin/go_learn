package main

// 逃逸分析
// go compiler 在编译的时候，会分析变量应该是在堆上分配还是在栈上分配
// go tool compile -m 可以查看变量逃逸结果

// 逃逸的两个总原则
// 1、指向栈对象的指针的生命期不能超过栈对象
// 2、指向栈对象的指针不能在堆上

import (
	"reflect"
)

// go tool compile -m escape.go
// output: value.go:10:17: i escapes to heap

func main() {
	i := 1
	reflect.ValueOf(&i) // 因为在 ValueOf() 内部调用了 escapes()函数，而 escapes 将指向i的地址赋给了全局变量 dump，所以 i 逃逸到堆上
}
