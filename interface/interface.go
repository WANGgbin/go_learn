package main

import "fmt"

// 注意接口的定义, 方法小写且接口和实现定义在同一个包中也是可以的.
type I interface {
	print() string
}

type t struct{}

func (p t) print() string {
	return "xxx"
}

func main() {
	var i I
	i = t{}
	fmt.Printf("%s", i.print())
}
