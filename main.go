package main

import (
	"fmt"

	"github.com/WANGgbin/golearn/internal"
)

func func1(i, j int64) int64 {
	return i + j
}

func func2(i, j, k int64) int64 {
	return i + j + k
}

func main() {
	fmt.Print(internal.AddWrapper(1, 2))
}
