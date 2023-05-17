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

// go for 循环，每次都定义一个新的变量
// 以下函数输出为：
// &i: 0xc000272788, &j: 0xc0002727a0
// &i: 0xc000272788, &j: 0xc0002727a8

//func main() {
//	for i := 0; i < 2; i++ {
//		j := 1
//		fmt.Printf("&i: %p, &j: %p\n", &i, &j)
//	}
//}
