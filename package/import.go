package main

/*

import 中的 '_' 和 '.' 代表什么含义呢?

_ 代表导入但不使用. 导入的目的仅仅是用来完成一些必要的初始化:全局变量, init() 函数.

. 代表将某个包中定义的 symbol 导入到当前文件中.
如果导入包跟当前文件中定义的符号冲突的话, 是报错还是优先使用当前文件中的符号呢?
*/

import (
	"fmt"

	. "github.com/WANGgbin/golearn/util"
)

// 当我们定义了 Add() 后, 编译的时候报错: Add redeclared in this block.
func Add(i, j int) int {
	return i + j
}

func main() {
	fmt.Print(Add(1, 2))
}
