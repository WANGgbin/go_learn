package main

import (
	"reflect"
)

type info struct {
}
type person struct {
	name string
	ptr  *info
}

func main() {
	p1 := person{name: "w", ptr: &info{}}
	p2 := person{name: "w", ptr: &info{}}

	print(p1 == p2)
	print(reflect.DeepEqual(p1, p2))
	println()
	print(&struct{}{})
	println()
	print(&struct{}{})
	println()
	print(&info{})
}
