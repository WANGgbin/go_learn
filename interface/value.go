package main

import (
	"reflect"
	"fmt"
)

// 注意：
// 通过反射修改值，一定要传递指针，如果传递的是值，则 Set 类函数设置的值的拷贝，因为这是没有意义的，所以会触发 panic。（通过检查 Value 中的类型元数据是不是指针类型）
// 传递指针，还需要调用 Elem() 函数，使得 Value() 中的类型元数据为值类型，而 data 还是指向原来的数据，从而能够修改值

func main() {
	i := 1
	val1 := reflect.ValueOf(i)
	val1.SetInt(2)
	fmt.Printf("i: %d\n", i)  // output: panic: reflect: reflect.Value.SetInt using unaddressable value
}

// func main() {
// 	i := 1
// 	val1 := reflect.ValueOf(&i)

// 	val2 := val1.Elem()
// 	val2.SetInt(2)

// 	fmt.Printf("i: %d\n", i)  // output: i: 2
// }

