package main

import (
	"fmt"
	"reflect"
)

type Person struct {

}
type Student struct {
	Person
}
func main() {
	for idx := 0; idx < reflect.TypeOf(Student{}).NumField(); idx++ {
		fieldTyp := reflect.TypeOf(Student{}).Field(idx)
		fmt.Printf("%+v", fieldTyp)
	}
}