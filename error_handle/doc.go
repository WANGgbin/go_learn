package error_handle

/*

本包用来描述 go 中，如何优雅的处理错误
1. fmt.Errorf("%w")
2. errors.As(err error, target any) bool
3. func Is(err, target error) bool
4. 应该仅仅处理一次错误，不要重复打印错误。一来代码不简洁，而来错误信息多，不好 debug.
5. 如果想显示忽略错误，请使用 _ 否则，很难分清楚是要忽略错误还是忘记了错误处理
6. 参考：https://github.com/xxjwxc/uber_go_guide_cn#%E9%94%99%E8%AF%AF%E7%B1%BB%E5%9E%8B
*/

func main() {

}
