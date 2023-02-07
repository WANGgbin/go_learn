package main

import "fmt"

/*
通常 go 里面接口的类型断言分为 4 类.
1. E -> 具体类型
2. I -> 具体类型
3. E -> I
4. I -> I
其中, E 表示空接口, I 表示非空接口

那么如何断定一个具体类型有没有实现某个接口呢? 比如 encoding/json 中的接口类型 Marshaler, 如果某个类型实现了该接口,则 json.Marshal 的时候会调用对应的 MarshalJSON 方法.
思路是通过将具体类型装箱到一个 E 中, 再通过 E -> I 的方式, 断言具体类型是否实现了 I.

我们可以看看 json 中的实现.

func marshalerEncoder(e *encodeState, v reflect.Value, opts encOpts) {
	// 最关键的就是这一步骤,转为一个 E,再进行类型断言
	m, ok := v.Interface().(Marshaler)
	if !ok {
		e.WriteString("null")
		return
	}
	b, err := m.MarshalJSON()
	...
}
*/

type myError struct {
	msg string
}

func (e *myError) Error() string {
	return e.msg
}

func isImplError(val interface{}) bool {
	_, ok := val.(error)
	return ok
}

func main() {
	e := &myError{msg: "something error"}
	ok := isImplError(e)
	if ok {
		fmt.Printf("type myError impl error interface")
	}
}
