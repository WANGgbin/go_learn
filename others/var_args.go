package main

import "fmt"

// 特别注意，可变参数的传递，需要打开后再传递

func f1(args ...interface{}) {
	fmt.Printf("%#v\n", args) //[]interface {}{"hi"}
	f2(args)                  // []interface {}{[]interface {}{"hi"}}
	f2(args...)               // []interface {}{"hi"}
}

func f2(args ...interface{}) {
	fmt.Printf("%#v\n", args)
}

func main() {
	f1("hi")
}
