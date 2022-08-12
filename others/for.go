package main

import "fmt"

// go 中,我们可以使用 for 来替代 while
func main() {
	i := 0
	for i < 10 {
		fmt.Printf("%d\n", i)
		i++
	}
}
