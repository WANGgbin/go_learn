package main

import "os"

func main() {
	os.OpenFile("main.go", os.O_RDONLY, 0666)
}