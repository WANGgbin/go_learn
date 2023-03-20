package main

import (
	"fmt"
	"reflect"
)

// 注意：
// 通过反射修改值，一定要传递指针，如果传递的是值，则 Set 类函数设置的值的拷贝，因为这是没有意义的，所以会触发 panic。（通过检查 Value 中的类型元数据是不是指针类型）
// 传递指针，还需要调用 Elem() 函数，使得 Value() 中的类型元数据为值类型，而 data 还是指向原来的数据，从而能够修改值

//func main() {
//	i := 1
//	val1 := reflect.ValueOf(i)
//	val1.SetInt(2)
//	fmt.Printf("i: %d\n", i)  // output: panic: reflect: reflect.Value.SetInt using unaddressable value
//}

// func main() {
// 	i := 1
// 	val1 := reflect.ValueOf(&i)

// 	val2 := val1.Elem()
// 	val2.SetInt(2)

// 	fmt.Printf("i: %d\n", i)  // output: i: 2
// }

/*
reflect.DeepEqual 的实现
本质上, 所有的复合类型的比较最后都转化为基本类型的比较.
*/

func myDeepEqual(dst, src any) bool {
	dVal := reflect.ValueOf(dst)
	sVal := reflect.ValueOf(src)

	if dVal.Type() != sVal.Type() {
		return false
	}

	return myDoDeepEqual(dVal, sVal)
}

func myDoDeepEqual(dst, src reflect.Value) bool {
	switch dst.Kind() {
	case reflect.Bool:
		return dst.Bool() == src.Bool()
	case reflect.Int8, reflect.Int16, reflect.Int, reflect.Int32, reflect.Int64:
		return dst.Int() == src.Int()
	case reflect.Uint8, reflect.Uint16, reflect.Uint, reflect.Uint32, reflect.Uint64:
		return dst.Uint() == src.Uint()
	case reflect.Float32, reflect.Float64:
		return dst.Float() == src.Float()
	case reflect.String:
		return dst.String() == src.String()
	case reflect.Slice:
		if dst.Len() != src.Len() {
			return false
		}
		// 对于 Slice, UnsafePointer() 返回第一个元素的地址。
		if dst.UnsafePointer() == src.UnsafePointer() {
			return true
		}
		for index := 0; index < dst.Len(); index++ {
			if !myDoDeepEqual(dst.Index(index), src.Index(index)) {
				return false
			}
		}
		return true
	case reflect.Array:
		for index := 0; index < dst.Len(); index++ {
			if !myDoDeepEqual(dst.Index(index), src.Index(index)) {
				return false
			}
		}
		return true
	case reflect.Struct:
		for index := 0; index < dst.NumField(); index++ {
			if !myDoDeepEqual(dst.Field(index), src.Field(index)) {
				return false
			}
		}
		return true
	case reflect.Map:
		if dst.Len() != src.Len() {
			return false
		}
		for _, key := range dst.MapKeys() {
			dstVal := dst.MapIndex(key)
			// MapIndex key 不存在的时候，返回 zero Value
			srcVal := dst.MapIndex(key)
			// IsValid() 用来判断一个 Value 是否是一个zero Value
			if !srcVal.IsValid() || !myDoDeepEqual(dstVal, srcVal) {
				return false
			}
		}
		return true
	case reflect.Interface:
		// Interface 的 IsNil() 判断 interface 是不是个 nil 值，即第一个 word 为 nil
		// 注意这种手法。
		if dst.IsNil() || src.IsNil() {
			return dst.IsNil() == src.IsNil()
		}
		return myDoDeepEqual(dst.Elem(), src.Elem())
	case reflect.Pointer:
		// 对于 Pointer, UnsafePointer() 返回 v.ptr
		if dst.UnsafePointer() == src.UnsafePointer() {
			return true
		}
		return myDoDeepEqual(dst.Elem(), src.Elem())

	case reflect.Func:
		// 注意函数的比较， 非空情况下，都是 false
		if dst.IsNil() && src.IsNil() {
			return true
		}
		return false
	}

	return false
}

//func main() {
//	type info struct {
//		self *info
//	}
//
//	i := info{}
//	i.self = &i
//
//	i2 := info{}
//	i2.self = &i2
//
//	// 我们自己的实现中，没有考虑这种情况，会导致无限递归，从而导致栈溢出: fatal error: stack overflow
//	myDeepEqual(i, i2)
//}

/*

reflect.Value 方法分类
reflect.Value 有两个重要的标志位置：
	flagIndir: Value.ptr 存储的是值本身还是指向值的指针。此标志是在 ValueOf() 方法中，根据 type.kind & kindDirectIface 来决定的。
	对于指针类型或者 unsafe.Pointer 类型，其类型元信息的 kind & kindDirectIface == 1
	flagAddr: 表示 Value.ptr 是否指向原始变量，如果设置 flagAddr, flagIndir 肯定也被设置。

1. Set 类方法
我们可以通过 Set*() 类方法来更改原始变量，那么什么时候才可以通过 Set() 方法来更改原始变量的值呢？

reflect.Value() 提供了一个 CanSet() 方法，用来判断能否修改 Value 的值。
func (v Value) CanSet() bool {
	// 只有存储了指向原始变量的地址 且 不是结构体的未导出字段，才是可以设置的。
	return v.flag&(flagAddr|flagRO) == flagAddr
}

当我们直接通过 reflect.ValueOf() 来获取一个变量对应的 Value 的时候，Value() 存储的都是原始变量的副本。
那么如果我们想要使用 Set() 方法修改原始变量应该怎么操作呢？

我们可以把变量的地址交给 ValueOf() 函数，同时结合 Elem() 方法来获取指向原始变量的 Value(). 此时便可以通过 Set() 方法
修改原始变量了。

func main() {
	i, j := 1, 2
	p := &i

	v := reflect.ValueOf(&p)
	fmt.Print(v.Elem().CanSet(), v.Elem().Kind()) // true, ptr
	v.Elem().Set(reflect.ValueOf(&j))

	fmt.Print(*p) // 2
}

除了 Elem() 方法， 切片类型的 Index() 方法也可以设置 flagAddr 标志。

func main() {
	s := []int{1, 2, 3}
	v := reflect.ValueOf(s)
	v.Index(0).SetInt(0)
	fmt.Print(s[0]) // 0
}

当一个 Value() 可以 Set 的时候，应该如何 Set 呢？ reflect 提供了一系列 Set() 方法。

// 注意！！！
// Set() 函数本质上就是内存的拷贝，注意跟深度拷贝的区别！！！
func (v Value) Set(x Value)
func (v Value) SetBool(x bool)
func (v Value) SetLen(n int)
func (v Value) SetCap(n int)
func (v Value) SetUint(x uint64)
func (v Value) SetString(x string)
...

按需使用，只需要注意，只有 Value() 指向原始变量的时候，才可以调用 Set*() 方法，否则会 panic !!!

2. New 类方法

reflect 提供了 New*() 方法 和 Make*() 方法来创建新的 Value.

New() 方法返回一个指向 typ 零值的 Pointer 类型 Value
func New(typ Type) Value

MakeSlice() 方法创建一个 0 值初始化的 长度为 len, 容量为 cap 的切片。
func MakeSlice(typ Type, len, cap int) Value

其他 Make*() 方法类似，不再赘述。
*/

/*

有了前面的基础知识，我们基于 reflect 实现一个 deepCopy.
核心思路: 先分配内存再进行赋值操作. 分配内存本质上调用的是 runtime 中的分配函数, 赋值操作跟具体的
类型有关.如果是基本类型,则直接拷贝即可.但对于引用类型(ptr, map, slice 等),则需要再分配内存并赋值.
整个过程有点类似 c++ 中的拷贝构造函数(根据一个已有的对象构造一个新的对象).

*/

// 注意！！！
// 目前，无法对结构体的未导出变量进行设置。
// 思路：New() 方法分配对象， Set() 方法进行拷贝
func myDeepCopy(src any) any {
	srcVal := reflect.ValueOf(src)
	return myDoDeepCopy(srcVal).Interface()
}

// 在我们的实现中，总是将 deepCopy 生成的 Value 复制给目标 Value, 这是因为 Set() 的本质
// 就是内存拷贝，如果不 deepCopy 直接使用 Set() 函数，是错误的！！！
func myDoDeepCopy(src reflect.Value) reflect.Value {
	var dst reflect.Value
	switch src.Kind() {
	case reflect.Pointer:
		elem := reflect.New(src.Type().Elem()).Elem()
		elem.Set(myDoDeepCopy(src.Elem()))
		return elem.Addr()
	case reflect.Struct:
		dst = reflect.New(src.Type()).Elem()
		for i := 0; i < src.NumField(); i++ {
			if !dst.Field(i).CanSet() {
				// 结构体的未导出字段不能更改
				panic(fmt.Sprintf("can't set field: %s", dst.Type().Field(i).Name))
			}
			dst.Field(i).Set(myDoDeepCopy(src.Field(i)))
		}
	case reflect.Slice:
		dst = reflect.MakeSlice(src.Type(), src.Len(), src.Cap())
		for i := 0; i < src.Len(); i++ {
			dst.Index(i).Set(myDoDeepCopy(src.Index(i)))
		}
	case reflect.Array:
		dst = reflect.New(src.Type()).Elem()
		for i := 0; i < src.Len(); i++ {
			dst.Index(i).Set(myDoDeepCopy(src.Index(i)))
		}
	case reflect.Map:
		dst = reflect.MakeMap(src.Type())
		// MapKeys() 返回的 []Value 是 src key 的副本 !!!
		for _, key := range src.MapKeys() {
			dst.SetMapIndex(key, myDoDeepCopy(src.MapIndex(key)))
		}
	default:
		// 对于基础类型，无须额外分配空间，直接返回 src
		return src
	}
	return dst
}

func main() {
	type sub struct {
		I int
	}
	type Info struct {
		S []int
		M map[int]int
		P *sub
	}

	info1 := &Info{
		S: []int{1, 2},
		M: map[int]int{1: 2, 2: 3},
		P: &sub{I: 1},
	}

	infoDeep := myDeepCopy(info1).(*Info)
	infoCopy := info1
	info1.S = append(info1.S, 3)
	info1.M[1] = 0
	info1.P.I = 0

	fmt.Printf("deep copy: s: %v, m: %v, p: %v\n", infoDeep.S, infoDeep.M, *infoDeep.P)
	fmt.Printf("copy: s: %v, m: %v, p: %v\n", infoCopy.S, infoCopy.M, *infoCopy.P)
}
