package main

import "os"
import "fmt"

func main() {
	os.OpenFile("main.go", os.O_RDONLY, 0666)
	fmt.Printf("hello world")
}
