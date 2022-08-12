package main

import (
	"reflect"
)

// reflect.Type 是个 interface
// *rtype 实现该接口，方法发的具体实现中，会将 *rtype 转化为对应的类型结构体，然后进行相应的操作。
// 如果某个类型不支持某种操作，则 panic


// common info
// common info 就是 rtype 定义如下

// rtype is the common implementation of most values.
// It is embedded in other struct types.
//
// rtype must be kept in sync with ../runtime/type.go:/^type._type.
// type rtype struct {
// 	size       uintptr
// 	ptrdata    uintptr // number of bytes in the type that can contain pointers
// 	hash       uint32  // hash of type; avoids computation in hash tables
// 	tflag      tflag   // extra type information flags
// 	align      uint8   // alignment of variable with this type
// 	fieldAlign uint8   // alignment of struct field with this type
// 	kind       uint8   // enumeration for C
// 	// function for comparing objects of this type
// 	// (ptr to object A, ptr to object B) -> ==?
// 	equal     func(unsafe.Pointer, unsafe.Pointer) bool
// 	gcdata    *byte   // garbage collection data
// 	str       nameOff // string form
// 	ptrToThis typeOff // type for pointer to this type, may be zero
// }


// uncommon type
// uncommon type 主要描述了类型的方法信息，定义如下

// uncommonType is present only for defined types or types with methods
// (if T is a defined type, the uncommonTypes for T and *T have methods).
// Using a pointer to this struct reduces the overall size required
// to describe a non-defined type with no methods.
// type uncommonType struct {
// 	pkgPath nameOff // import path; empty for built-in types like int, string
// 	mcount  uint16  // number of methods
// 	xcount  uint16  // number of exported methods
// 	moff    uint32  // offset from this uncommontype to [mcount]method
// 	_       uint32  // unused
// }


// 这里单独列举这个方法的目的是为了学习两个技巧
// 1、在切片的时候，在控制 len 的同时，如何设置 cap？ 方法 [:len:cap]
// 2、现在只知道第一个method 的地址，如何将其转化为一个 slice，方法就是 先强转为数组指针，然后通过截取数组返回一个 slice

// func (t *uncommonType) methods() []method {
// 	if t.mcount == 0 {
// 		return nil
// 	}
// 	return (*[1 << 16]method)(add(unsafe.Pointer(t), uintptr(t.moff), "t.mcount > 0"))[:t.mcount:t.mcount]
// }

