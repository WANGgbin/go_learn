package error_handle

/*

本包用来描述 go 中，如何优雅的处理错误
1. fmt.Errorf("%w")

目的是不断链，这样 caller 可以使用 As()/Is() 来匹配错误值或者匹配错误类型。

2. errors.As(err error, target any) bool
3. errors.Is(err, target error) bool

这两个方法依赖于错误的 UnWrap() 方法，大多数时候使用 fmt.Errorf(%w) 即可。

4. 应该仅仅处理一次错误，不要重复打印错误。一来代码不简洁，而来错误信息多，不好 debug.

个人建议在第一次错误的地方打印 log，这样方便排查错误。

5. 如果想显式忽略错误，请使用 _ 否则，很难分清楚是要忽略错误还是忘记了错误处理

6. 错误命名

var 以 Err/err 开头
错误类型以 Error 结尾

7. 参考：https://github.com/xxjwxc/uber_go_guide_cn#%E9%94%99%E8%AF%AF%E7%B1%BB%E5%9E%8B

*/

func main() {

}
