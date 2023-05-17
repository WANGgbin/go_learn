package main

/*

动态类型循环引用的实现. 即先生成一个动态类型, 然后动态类型内部某个成员指向该动态类型.
方法是通过 reflect 中定义的一些结构体, 来直接修改动态类型的类型元数据.

*/
import (
	"encoding/json"
	"fmt"
	"reflect"
	"unsafe"
)

type Person struct {
	Self *Person
	Name string
	Age  int
}

func (p *Person) GetName() string {
	return p.Name
}

func (p *Person) GetAge() int {
	return p.Age
}

type tflag uint8
type nameOff int32 // offset to a name
type typeOff int32 // offset to an *rtype
type textOff int32
type rtype struct {
	size       uintptr
	ptrdata    uintptr // number of bytes in the type that can contain pointers
	hash       uint32  // hash of type; avoids computation in hash tables
	tflag      tflag   // extra type information flags
	align      uint8   // alignment of variable with this type
	fieldAlign uint8   // alignment of struct field with this type
	kind       uint8   // enumeration for C
	// function for comparing objects of this type
	// (ptr to object A, ptr to object B) -> ==?
	equal     func(unsafe.Pointer, unsafe.Pointer) bool
	gcdata    *byte   // garbage collection data
	str       nameOff // string form
	ptrToThis typeOff // type for pointer to this type, may be zero
}

type name struct {
	bytes *byte
}

func add(p unsafe.Pointer, x uintptr, whySafe string) unsafe.Pointer {
	return unsafe.Pointer(uintptr(p) + x)
}

func (n name) data(off int, whySafe string) *byte {
	return (*byte)(add(unsafe.Pointer(n.bytes), uintptr(off), whySafe))
}

func (n name) readVarint(off int) (int, int) {
	v := 0
	for i := 0; ; i++ {
		x := *n.data(off+i, "read varint")
		v += int(x&0x7f) << (7 * i)
		if x&0x80 == 0 {
			return i + 1, v
		}
	}
}

func (n name) name() (s string) {
	if n.bytes == nil {
		return
	}
	i, l := n.readVarint(1)
	hdr := (*reflect.StringHeader)(unsafe.Pointer(&s))
	hdr.Data = uintptr(unsafe.Pointer(n.data(1+i, "non-empty string")))
	hdr.Len = l
	return
}

type structField struct {
	name        name    // name is always non-empty
	typ         *rtype  // type of field
	offsetEmbed uintptr // byte offset of field<<1 | isEmbedded
}

type structType struct {
	rtype
	pkgPath name
	fields  []structField // sorted by offset
}

func main() {
	newTyp := reflect.StructOf([]reflect.StructField{
		{
			Name: "Person",
			Type: reflect.TypeOf(&Person{}),
		},
		{
			Name: "Extra",
			Type: reflect.TypeOf(1),
		},
	})

	newVal := reflect.New(newTyp).Elem()
	newPointer := reflect.New(newTyp)

	newValType := (*structType)(*((*unsafe.Pointer)(unsafe.Pointer(&newVal))))
	newPointerType := *((*unsafe.Pointer)(unsafe.Pointer(&newPointer)))

	for index, field := range newValType.fields {
		if field.name.name() == "Person" {
			newValType.fields[index].typ = (*rtype)(newPointerType)
		}
	}

	newVal.FieldByName("Person").Set(reflect.New(newTyp))
	info, err := json.Marshal(newVal.Interface())
	if err != nil {
		panic(err)
	}

	fmt.Printf("%s\n", string(info))

}

// func main() {
// 	p := &Person{}
// 	p1 := &Person{}
// 	p.Self = p1
// 	info, err := json.Marshal(p)
// 	if err != nil {
// 		panic(err)
// 	}
// 	fmt.Printf("%s\n", string(info))
// }
