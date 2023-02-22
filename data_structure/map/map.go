package main

import (
	"encoding/json"
)

/*
	map 也是引用类型，map 赋值之后还是引用相同的数据结构
*/

//func main() {
//	m := map[int]int{
//		1: 1,
//	}
//
//	m1 := m
//	m1[1] = 2
//
//	//  m: map[1:2]
//	//	m1:map[1:2]
//	fmt.Printf("m: %v\nm1:%v\n", m, m1)
//
//	// 注意下面两种操作改变了 m1 的指向，但是并不会影响 m 的指向
//	m1 = nil
//	// m: map[1:2]
//	// m1:map[]
//	fmt.Printf("m: %v\nm1:%v\n", m, m1)
//
//	m1 = map[int]int{1: 3}
//	// m: map[1:2]
//	// m1:map[1:3]
//	fmt.Printf("m: %v\nm1:%v\n", m, m1)
//}

/*
	但是在 json.Unmarshal 的时候， map 类型的变量是必须要传递地址的。map 不是引用的相同的底层数据吗？为什么还要传递 map 变量的地址呢？
主要考虑一种特殊情况，如果传递的是 nil map， Unmarshal 内部会新建一个 map，所以如果只传递 map 值对象 是无法更改原来 map 的值的。
*/

func main() {
	var m map[int]int
	jsonStr := []byte("{\"1\":1}")
	//err := json.Unmarshal(jsonStr, m)
	//if err != nil {
	//	panic(err) // json: Unmarshal(non-pointer map[int]int)
	//}

	err := json.Unmarshal(jsonStr, &m)
	if err != nil {
		panic(err) // nil
	}
}
