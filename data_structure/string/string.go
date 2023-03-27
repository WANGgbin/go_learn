package main

import (
	"fmt"
)

/*

字符串截取
*/
// func main() {
// 	s := "abc"
// 	s1 := s[3:3] // 字符串切片的取值范围: 0<= i <= j <= len(str)
// 	fmt.Printf("%s\n", s1)
// }

/*

字符串编码

unicode 跟 utf-8 的区别是什么呢?
unicode 是一种字符集合, 定义了每个字符对应的数值是什么. 比如 "中" 就是 0x4e2d. 这个数值就是所谓
的码点(unicode point), 在 go 中对应的就是 rune, 而 rune 是 int32 的别名.

那么数值如何在计算机中存储呢? 这就是 utf-8 的作用, 通过变长的方式解决了存储空间浪费的问题.

0xxxxxxx
110xxxxx 10xxxxxx
1110xxxx 10xxxxxx 10xxxxxx
11110xxx 10xxxxxx 10xxxxxx 10xxxxxx

*/

// func main() {
// 	r := []rune("中国")
// 	fmt.Printf("%#x", r)

// 	var builder strings.Builder
// 	builder.WriteRune(r[0])
// 	str := builder.String()

// 	fmt.Printf("%d", len(str))
// }

/*

字符串常量

字符串常量中如何嵌入八进制, 十六进制 以及 码点 呢?
八进制: \000 不能超过 \3777
十六进制: \xhh 必须是两位 16 进制数字
码点: \uhhhh 四位 16 进制数字

*/
// func main() {
// 	str1 := "\u4e2d\u56fd" // 会将码点转化为 utf-8
// 	fmt.Print(str1) // 输出 "中国".
// 	fmt.Print("\n")
// }

/*

字符串遍历

go 中字符串的遍历是按照 rune 走的
*/

func main() {
	str1 := "\u4e2d\u56fd"
	// 输出:
	// 中
	// 国
	for _, r := range str1 {
		fmt.Print(string(r))
		fmt.Print("\n")
	}
}
