package main

import (
	"fmt"
	"reflect"
	"unsafe"
)

// 介绍 []byte 与 string 的转化

// 最简单的方式是:
// []byte -> string: string([]byte)
// string -> []byte: []byte(string)

// 但前面的方式涉及到内存重新分配,如果涉及到大量的 []byte 和 string 的转化,则考虑使用下面的方式,参考 Gin 的 bytesconv 包

// string -> []byte without memory allocation
// 注意,转化为 []byte 后,如果修改 []byte 可能会触发 SIGSEGV 错误(如果 string 在 .rdata 区,触发 SIGSEGV 错误,如果 string
// 在堆区,则不会发生错误(string 运行时拼接会分配到堆区))
// 当然 string 转化为 []byte 后不要修改内容
func String2Bytes(str string) []byte {
	strPtr := (*reflect.StringHeader)(unsafe.Pointer(&str))
	return *(*[]byte)(unsafe.Pointer(&reflect.SliceHeader{Data: strPtr.Data, Len: strPtr.Len, Cap: strPtr.Len}))
}

// func main() {
// 	str := "abcd"
// 	str1 := str + "efg" // 运行时分配 str1
// 	bytes := String2Bytes(str1)
// 	bytes[0] = 101  // 修改不会发生错误
// 	fmt.Printf("%v", bytes)
// }

// func main() {
// 	str := "abcd"
// 	bytes := String2Bytes(str)
// 	bytes[0] = 101  // unexpected fault address 0x495792
// 					// fatal error: fault
// 					// [signal SIGSEGV: segmentation violation code=0x2 addr=0x495792 pc=0x47f7e5]
// 	fmt.Printf("%v", bytes)
// }


func Bytes2String(b []byte) string {
	bPtr := (*reflect.SliceHeader)(unsafe.Pointer(&b))
	return *(*string)(unsafe.Pointer(&reflect.StringHeader{Data: bPtr.Data, Len: bPtr.Len}))
}

func main() {
	b := []byte{98, 99, 100}
	str := Bytes2String(b)
	fmt.Printf("%s\n", str)  // bcd
}