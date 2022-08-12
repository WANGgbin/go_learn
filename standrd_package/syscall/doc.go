package main

// 标准包 syscall 学习

import (
	"fmt"
	"syscall"
)

// 创建一个文件并写入 "你好 go syscall!"

func main() {
	fd, err := syscall.Open("./hello.txt", syscall.O_CREAT|syscall.O_WRONLY, uint32(0666))
	if err != nil {
		fmt.Printf("Open file hello.txt error: %v\n", err)
	} else {
		write_len, err := syscall.Write(fd, []byte("你好 go syscall!"))
		if err != nil {
			fmt.Printf("Write file error: %v\n", err)
		} else {
			fmt.Printf("write %d byte in file\n", write_len)
		}
	}
	syscall.Close(fd)
}

// go 的 syscall包 本质上就是通过汇编指令SYSCALL 陷入内核. 内核系统调用之返回一个整型,
// 如果系统调用出错,则返回一个表示具体错误信息的负值. syscall 包在系统调用出错的时候,会
// 将负值包装为一个 err 并返回.
// 在 c 的标准库中,会将错误吗赋值到 errno 全局变量中. 本质上不管是 go 还是 c 调用系统
// 调用的方式都是一样的.
