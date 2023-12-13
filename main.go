package main

import (
	"fmt"
	"reflect"
)

type Person struct {
	Age int
}

func (p *Person) SetAge(param int) {
	p.Age = param
}

type Student struct {
	Person
}

type myInt int

func (m *myInt) Set(p int) {
	*m = myInt(p)
}

func add(inputs []reflect.Value) []reflect.Value {
	var ret int64
	for _, input := range inputs {
		ret += input.Int()
	}

	return []reflect.Value{reflect.ValueOf(ret)}
}

func main() {

	fn := reflect.MakeFunc(reflect.TypeOf((func(i, j int) int64)(nil)), add)

	ret := fn.Interface().(func(int, int) int64)(1, 2)
	fmt.Printf("%d\n", ret)

	rets := fn.Call([]reflect.Value{reflect.ValueOf(1), reflect.ValueOf(2)})
	// fmt.Printf("%d\n", rets[0].Int())
}
