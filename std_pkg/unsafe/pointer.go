package main

/**
unsafe.Pointer 与 uintptr 的区别是什么呢?

运算方面:
unsafe.Pointer 不可以直接进行指针运算, 需要转化为 uintptr, 才能进行指针运算.

gc 方面:
unsafe.Pointer 标志这指向一个对象, gc 扫描的时候, 会标记指向的对象.
而 uintptr 本质就是个 uint 其长度同平台的指针长度一致,如果使用 uintptr 指向一个对象,则会造成
对象无法被标记进而被回收导致错误.
*/

import (
	"fmt"
	"unsafe"
)

func main() {

	arr := [2]uint32{1, 2}

	ptr1 := &arr[0]

	// 注意: unsafe.Pointer 转化为 *uint32 的时候一定要将 *uint32 扩起来,否则 go
	// 会认为是将 unsafe.Pointer 先转化为 int32, 然后再解引用.
	// ptr1 = *uint32(unsafe.Pointer(...))   wrong!
	// ptr1 = (*uint32)(unsafe.Pointer(...)) right!
	ptr1 = (*uint32)(unsafe.Pointer(uintptr(unsafe.Pointer(ptr1)) + unsafe.Sizeof(uint32(0))))

	fmt.Println(*ptr1)
}
