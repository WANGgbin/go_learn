本文描述 go 泛型编程。

# 使用

## 特化

不支持


## go 类型泛型
## go receiver 泛型
## go method 泛型

为什么不支持泛型？

实现困难，参考：https://github.com/golang/proposal/blob/master/design/43651-type-parameters.md#no-parameterized-methods

## go func 泛型

## 使用场景

# 性能

# 原理

## 参考资源：
- [go 泛型](https://www.cnblogs.com/taoxiaoxin/p/17933517.html)
- [简单易懂的 Go 泛型使用和实现原理介绍](https://zhuanlan.zhihu.com/p/509290914)
- [a-gentle-introduction-to-generics-in-go/](https://dominikbraun.io/blog/a-gentle-introduction-to-generics-in-go/) // 上述文章的原文
- [Go 1.18 泛型全面讲解](https://juejin.cn/post/7080938405449695268#heading-29) 

# 注意事项

- 泛型函数中不能**直接**访问泛型类型的属性

```go
type MyStruct1 struct{}

func (m MyStruct1) Method1() {
	fmt.Printf("MyStruct1::Method1")
}

type MyStruct2 MyStruct1

func (m MyStruct2) Method1() {
	fmt.Printf("MyStruct2::Method1")
}

type Param interface {
	Method1()
}

func Print1[T MyStruct1|MyStruct2](obj T) {
	obj.Method1() // 报错：obj.Method1 undefined (type T has no field or method Method1)
}
```

可以通过**接口**的方式来调用 Method1() 方法：

```go
type Param interface {
	Method1()
}

func Print[T Param](obj T) {
	obj.Method1()
}

func main() {
	var obj MyStruct1
	Print(obj)

	var obj1 MyStruct2
	Print(obj1)
}
```
