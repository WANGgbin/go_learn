package main

import (
	"fmt"
	_ "unsafe"
	_ "local.com/test/utils"
)

//go:linkname utils_add local.com/test/utils.add
func utils_add(a, b int64) int64

func main() {
	fmt.Printf("%d plus %d equals %d\n", 1, 2, utils_add(1, 2))
}
