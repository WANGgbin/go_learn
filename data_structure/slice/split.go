package main

// 可以通过 source[:len:cap] 的方式来控制生成的切片的长度和容量
// func main() {
// 	s1 := make([]int, 0, 10)

// 	// output: len: 2, cap: 10
// 	s2 := s1[:2]
// 	fmt.Printf("len: %d, cap: %d\n", len(s2), cap(s2))

// 	// output: len: 2, cap: 2
// 	s3 := s1[:2:2]
// 	fmt.Printf("len: %d, cap: %d\n", len(s3), cap(s3))
// }

// 还需要注意索引的取值范围 0 <= index <= capacity 同时 low <= high
//func main() {
//	s1 := make([]int, 0, 2)
//	//s2 := s1[2:1] // Invalid index values, must be low <= high
//	//s2 := s1[2:3] //  slice bounds out of range [:3] with capacity 2
//	s2 := s1[2:2] // success, len: 0, cap: 0
//	fmt.Printf("len: %d, cap: %d\n", len(s2), cap(s2))
//}

// go slice 是不支持切片的 + 操作的,只能通过 append() 扩展切片
// func main() {
// 	s1 := []int{1, 2}
// 	s2 := []int{3, 4}
// 	s3 := append(s1, s2...)
// 	fmt.Printf("%v", s3)
// }

// go string 支持 + 操作
// func main() {
// 	// s1 := "abcd"
// 	// s2 := "ef"
// 	// s3 := s1 + s2
// 	// fmt.Printf("%s", s3)

// 	s1 := "abcd"
// 	s2 := s1[:2]
// 	s3 := s1[3:]
// 	s4 := s2 + s3
// 	fmt.Printf("%s", s4)
// }
