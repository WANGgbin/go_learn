package main

import (
	"bufio"
	"fmt"
)

// bufio 本质上就是加了一个中间层,来提高效果,避免多次的底层输入/输出.类似于 C 标准I/O的缓冲区
// bufio.Reader 实现 io.Reader 接口
// bufio.Writer 实现 io.Writer 接口

type WriteNothing struct{}

func (w WriteNothing) Write(data []byte) (writen int, err error) {
	fmt.Printf("write: %s\n", string(data))
	return len(data), nil
}

func main() {
	w := bufio.NewWriterSize(WriteNothing{}, 4)
	fmt.Printf("size: %d\n", w.Size())
	for i := 0; i < 5; i++ {
		w.WriteByte(byte(i))
		fmt.Printf("avaliable: %d\n", w.Available())
		fmt.Printf("buffered: %d\n", w.Buffered())
	}
}
