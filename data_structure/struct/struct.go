package main

/*
	go 内部的空结构体是怎么实现的？
	空结构体的大小到底是多少呢？ 0. 通过类型元信息：rtype.size 即可判断

	空结构体其实可以看作是一个常量，对于这样的变量，其值是无法改变的。且所有的空结构体变量都一样。因此
在分配空结构体变量的时候，没有必要为每一个空结构体都分配一部分内存。

	因此，go 提供了一个 runtime.zerobase 变量，当为所有 size == 0 的对象分配内存的时候，实际上并没有分配内存，
仅仅是返回 &runtime.zerobase。

	我们看看 mallocgc() 中的部分源码：
```go
var zerobase uintptr

func mallocgc(size uintptr, typ *_type, needzero bool) unsafe.Pointer {

	if size == 0 {
		return unsafe.Pointer(&zerobase)
	}
}
```

	但情况并不总是这样。
	当空结构体作为一个大的结构体的非第一个字段的时候(如果是第一个字段，没有必要分配空间，此时结构体本身的地址就是空结构体字段的地址)，
就需要为该字段单独分配存储空间。那么空间大小是多少呢？ 1 字节。

*/

func main() {

}