本文介绍 `reflect.Type` 相关内容。

# rtype 结构体
reflect.Type 是个 interface
*rtype 实现该接口，方法发的具体实现中，会将 *rtype 转化为对应的类型结构体，然后进行相应的操作。
如果某个类型不支持某种操作，则 panic

common info
common info 就是 rtype 定义如下
```go
// rtype is the common implementation of most values. It is embedded in other struct types.

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
```
uncommonType is present only for defined types or types with methods
(if T is a defined type, the uncommonTypes for T and *T have methods).
Using a pointer to this struct reduces the overall size required
to describe a non-defined type with no methods.
type uncommonType struct {
	pkgPath nameOff // import path; empty for built-in types like int, string
	mcount  uint16  // number of methods
	xcount  uint16  // number of exported methods
	moff    uint32  // offset from this uncommontype to [mcount]method
	_       uint32  // unused
}

这里单独列举这个方法的目的是为了学习两个技巧
1、在切片的时候，在控制 len 的同时，如何设置 cap？ 方法 [:len:cap]
2、现在只知道第一个method 的地址，如何将其转化为一个 slice，方法就是 先强转为数组指针，然后通过截取数组返回一个 slice
```go
func (t *uncommonType) methods() []method {
	if t.mcount == 0 {
		return nil
	}
	return (*[1 << 16]method)(add(unsafe.Pointer(t), uintptr(t.moff), "t.mcount > 0"))[:t.mcount:t.mcount]
}
```

# method/imethod/Method/FuncType

- method

method 表示的是自定义类型的方法类型。我们看看 method 的定义：
```go
type method struct {
	name nameOff // name of method
	mtyp typeOff // 方法对应的函数类型 (without receiver)
	ifn  textOff // fn used in interface call (one-word receiver)
	tfn  textOff // fn used for normal method call
}
```
ifn 和 tfn 什么区别呢？通过接口调用对象的函数在传递对象的时候，怎么确定对象的大小呢？一种方式可以从类型元信息中
获取类型的大小，但这样效率太低。为此，go 编译器基于对象原来的函数 tfn 构造了一个接受指针类型的函数，这样在调用
对象方法的时候，只需要传递对象指针即可。

举个例子，假如有以下定义：
```go
type person struct {
	name string
}

// tfn
func (p person) getName() string {
	return p.name
}

// ifn
func (p *person) getName() string {
	return (*p).getName()
}
```

另外需要注意，**tfn/ifn 实际上是闭包对象的地址，并不是函数的真正地址**

- imethod

imethod 表示的是接口中定义的方法，没有具体的值。定义如下：

```go
type imethod struct {
	name nameOff // name of method
	typ  typeOff // 函数类型
}
```

这里我们就可以看看 `Type.Implements(u Type) bool` 的实现。

```go
// 本质上，就是判断对象类型的 method[] 数组有没有覆盖接口的 imethod[] 数组
// implements reports whether the type V implements the interface type T.
func implements(T, V *rtype) bool {
	// T 必须是接口类型
	if T.Kind() != Interface {
		return false
	}
	// 如果 T 是空接口，直接返回 true
	t := (*interfaceType)(unsafe.Pointer(T))
	if len(t.methods) == 0 {
		return true
	}

	// The same algorithm applies in both cases, but the
	// method tables for an interface type and a concrete type
	// are different, so the code is duplicated.
	// In both cases the algorithm is a linear scan over the two
	// lists - T's methods and V's methods - simultaneously.
	// Since method tables are stored in a unique sorted order
	// (alphabetical, with no duplicate method names), the scan
	// through V's methods must hit a match for each of T's
	// methods along the way, or else V does not implement T.
	// This lets us run the scan in overall linear time instead of
	// the quadratic time  a naive search would require.
	// 如果 v 是接口类型，则需要看 v 的 imethod[] 是否覆盖 t 的 imethod[] 数组
	if V.Kind() == Interface {
		v := (*interfaceType)(unsafe.Pointer(V))
		i := 0
		for j := 0; j < len(v.methods); j++ {
			tm := &t.methods[i]
			tmName := t.nameOff(tm.name)
			vm := &v.methods[j]
			vmName := V.nameOff(vm.name)
			if vmName.name() == tmName.name() && V.typeOff(vm.typ) == t.typeOff(tm.typ) {
				// 如果未导出，需要在同一个包。
				if !tmName.isExported() {
					tmPkgPath := tmName.pkgPath()
					if tmPkgPath == "" {
						tmPkgPath = t.pkgPath.name()
					}
					vmPkgPath := vmName.pkgPath()
					if vmPkgPath == "" {
						vmPkgPath = v.pkgPath.name()
					}
					if tmPkgPath != vmPkgPath {
						continue
					}
				}
				// 覆盖 T 所有的 imethod
				if i++; i >= len(t.methods) {
					return true
				}
			}
		}
		return false
	}
	
	// 对于具体类型对象，如果没有方法直接返回 false
	v := V.uncommon()
	if v == nil {
		return false
	}
	i := 0
	vmethods := v.methods()
	// 遍历 v 的每一个方法。
	for j := 0; j < int(v.mcount); j++ {
		tm := &t.methods[i]
		tmName := t.nameOff(tm.name)
		vm := vmethods[j]
		vmName := V.nameOff(vm.name)
		// 只有名字和类型元信息相同，才认为 method 实现了 imethod
		if vmName.name() == tmName.name() && V.typeOff(vm.mtyp) == t.typeOff(tm.typ) {
			// 与上面代码完全一致
		}
	}
	return false
}
```

- Method

method 与 Method 的区别是什么呢？

method 是类型元信息的一部分，而 Method 是应用层面的内容。定义如下：
```go
type Method struct {
	// Name is the method name.
	Name    string
	// 只有方法未导出，PkgPath 才非空。
	PkgPath string

	Type  Type  // method type
	Func  Value // func with receiver as first argument
	Index int   // index for Type.Method
}
```

Method.Type 与 method.mtyp 一样吗？

Method.Type 是携带 receiver 的函数类型，而 method.mtyp 是未携带 receiver 的函数类型。还是以前面的
person 类型的 getName() 方法举例。

method.mtyp 对应的函数类型为：`func () string`，Method.Type 对应的函数类型为：`func (person) string`

我们来看看 `Type.Method(int) Method` 的实现。

```go
func (t *rtype) Method(i int) (m Method) {
	methods := t.exportedMethods()
	p := methods[i]
	pname := t.nameOff(p.name)
	m.Name = pname.name()
	fl := flag(Func)
	mtyp := t.typeOff(p.mtyp)
	ft := (*funcType)(unsafe.Pointer(mtyp))
	in := make([]Type, 0, 1+len(ft.in()))
	// 第一个参数类型为 t
	in = append(in, t)
	for _, arg := range ft.in() {
		in = append(in, arg)
	}
	out := make([]Type, 0, len(ft.out()))
	for _, ret := range ft.out() {
		out = append(out, ret)
	}
	// 动态构造类型元信息
	mt := FuncOf(in, out, ft.IsVariadic())
	m.Type = mt
	// 这里进一步说明 p.tfn 是个闭包对象
	tfn := t.textOff(p.tfn)
	fn := unsafe.Pointer(&tfn)
	m.Func = Value{mt.(*rtype), fn, fl}

	m.Index = i
	return m
}
```

- funcType

funcType 就是真正的函数类型，method/imethod/Method 中的 type 指向的就是 funcType. 定义如下：
```go
type funcType struct {
	rtype
	inCount  uint16
	outCount uint16 // top bit is set if last input parameter is ...
}
```

funcType 后面跟着每一个入参/出参的 类型元信息。

# 构造动态类型

我们看看 go 内部类型的动态生成. 除了编译时生成的类型元信息, go 还提供了运行时生产类型元信息, 存储在堆中.

```go
func main() {
	fields := []reflect.StructField{
		{
			Name:      "Age",
			Type:      reflect.TypeOf(1),
			Tag:       `json:"age"`,
			Anonymous: false,
		},
		{
			Name:      "Name",
			Type:      reflect.TypeOf(""),
			Tag:       `json:"name"`,
			Anonymous: true,
		},
	}
	/*

		type person struct {
			name string
			age  int
		}

		func (p *person) GetName() string {
			return p.name
		}

		func (p *person) GetAge() int {
			return p.age
		}

		func main() {
			p := &person{name: "wgb", age: 12}
			v := reflect.ValueOf(p)

			name := v.Method(0).Call([]reflect.Value{})[0].Interface().(string)
			print(name)

			/*

			动态构造一个 Struct, 此外还可以调用 SliceOf, ChanOf, MapOf 来动态构造类型
	*/
	typ := reflect.StructOf(fields)

	fmt.Printf("name: %d\n", typ.Size())
}
```