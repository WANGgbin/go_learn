package main

// 注意接口的定义, 方法小写且接口和实现定义在同一个包中也是可以的.
//type I interface {
//	print() string
//}
//
//type t struct{}
//
//func (p t) print() string {
//	return "xxx"
//}
//
//func main() {
//	var i I
//	i = t{}
//	fmt.Printf("%s", i.print())
//}

/*
interface{} 装箱/拆箱
*/

func main() {
	// 注意 slice 类型装箱，修改原来的 slice，会影响到箱中的 slice
	//s := []int{1, 2, 3}
	//var i interface{} = s
	//s[0] = 0
	//fmt.Print(i.([]int)[0])

}
