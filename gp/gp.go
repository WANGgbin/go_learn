package main

import "fmt"

type MyStruct1 struct{}

func (m MyStruct1) Method1() {
	fmt.Printf("MyStruct1::Method1")
}

type MyStruct2 MyStruct1

func (m MyStruct2) Method1() {
	fmt.Printf("MyStruct2::Method1")
}

type Param interface {
	Method1()
}

func Print1[T MyStruct1 | MyStruct2](obj T) {
	obj.Method1()
}

func Print[T Param](obj T) {
	obj.Method1()
}

func main() {
	var obj MyStruct1
	Print(obj)

	var obj1 MyStruct2
	Print(obj1)
}
