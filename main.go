package main

import (
	"fmt"
	"strings"
	"unsafe"
)

func func1(i, j int64) int64 {
	return i + j
}

func func2(i, j, k int64) int64 {
	return i + j + k
}

func main() {
	var v struct{}
	var b bool
	fmt.Printf("%d\n", unsafe.Sizeof(v))
	fmt.Printf("%d\n", unsafe.Sizeof(b))

	var list1 [10]struct{}
	var list2 [10]bool
	fmt.Printf("%d\n", unsafe.Sizeof(list1))
	fmt.Printf("%d\n", unsafe.Sizeof(list2))
	strings.Builder()	
}
