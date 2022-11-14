package main

// go 数组若干重要的特性

//数组变量之间可以直接赋值,函数传递也是值拷贝. 与其他语言不一样的是,数组变量并不表示第一个元素的地址,
//而表示整个数组.
// func handleArr(arr [2]int) {
// 	fmt.Printf("%p\n", &arr)
// }
// func main() {
// 	arr := [2]int{1, 2}
// 	arr1 := arr
// 	// 0xc000134020
// 	// 0xc000134000
// 	// 0xc000134010
// 	handleArr(arr)
// 	fmt.Printf("%p\n", &arr)
// 	fmt.Printf("%p\n", &arr1)
// }

// 数组有一种特殊的初始化语句,类似 map, key 是索引, key 的顺序可以是任意的,未赋值的元素,值为元素类型
// 对应的零值.在创建大小固定且下标是整型且连续的 map 的时候,可以考虑使用数组,数组本身就是个哈系表.一个
// 典型的例子是 syscall.signals 的定义:
// var signals = [...]string{
// 	1: "hangup",
// 	2: "interrupt",
// 	...
// }

// func main() {
// 	arr := [...]string{
// 		1: "1",
// 		2: "2",
// 	}

// 	fmt.Printf("%d\n", len(arr)) // 3
// 	fmt.Printf("%s\n", arr[0])  // ""
// }

// 数组指针的特殊操作
// func zero(arr *[2]int) {
// 	// 注意这种遍历数组的方式
// 	for index := range arr {
// 		arr[index] = 0
// 	}
// }

// func zero1(arr *[2]int) {
// 	for index, value := range arr {
// 		if value != 0 {
// 			arr[index] = 0
// 		}
// 	}
// }

// func zero2(arr *[2]int) {
// 	*arr = [2]int{}
// }

// func main() {
// 	arr := [2]int{1, 2}
// 	zero(&arr)
// 	fmt.Printf("%v\n", arr) // [0 0]

// 	arr = [2]int{1, 2}
// 	zero1(&arr)
// 	fmt.Printf("%v\n", arr) // [0 0]

// 	arr = [2]int{1, 2}
// 	zero2(&arr)
// 	fmt.Printf("%v\n", arr) // [0 0]

// }

//只要数组中的元素是可比较的,那么数组也是可比较的.
// func main() {
// 	arr := [2]int{1, 2}
// 	arr1 := [2]int{1, 2}

// 	fmt.Printf("%t\n", arr == arr1) // true

// 	arr2 := [3]int{1}
// 	fmt.Printf("%t\n", arr == arr2) //  (mismatched types [2]int and [3]int)
// }
