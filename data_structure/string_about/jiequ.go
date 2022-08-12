package main

import "fmt"

func main() {
	s := "abc"
	s1 := s[3:3] // 字符串切片的取值范围: 0<= i <= j <= len(str)
	fmt.Printf("%s\n", s1)
}
