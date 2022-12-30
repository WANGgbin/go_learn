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


