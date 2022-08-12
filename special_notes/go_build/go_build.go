package main

/*
	go 中可以使用特别注释"//go:build os arch" 或者"// +build os arch" 来实现跨平台代码
 第一种方式从 go1.17 开始支持,注意同其他特别注释一样第一种方式的//后面是没有空格的,但是第二种
 方式// 后面必须要有一个空格.
	此类注释是以文件为单位的,也就是说跨平台编译的时候,是以文件为单位选择性编译的.必须位于 package 声明之前
且必须与 package 声明之间要有一个空格.
	第一种方式的条件表达式支持 || &&,比如: //go:build (os || windows) && amd64
	第二种表达式,逗号表示与,空格表示或,感叹号表示非,比如: // +build os windows

	条件编译除了使用注释外,还可以通过文件命名的方式比如:filename_os_arch.go 这样 go 就会选择对应的文件进行编译
	通过测试我们发现, go 是通过读取 go env 来获取对应的 OS 和 ARCH 的.
*/

func main() {
	Print()
	add()
}
