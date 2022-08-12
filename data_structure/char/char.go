package main

// go 中 将字符放在一对单引号内表示一个字符常量,常量值为对应的utf-8码点值
// func main() {
// 	// c := ':'
// 	b := byte(1)
// 	fmt.Printf("%v", '中' == b)  // 这里'中'对应的码点值为 20013, 所以报错:constant 20013 overflows byte
// }

// 如果字符常量赋值给一个未制定类型变量,则变量类型为 rune
// func main() {
// 	c := ':'  // c: rune, (type rune=int32)
// }
