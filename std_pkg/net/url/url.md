描述 net/url 的使用

# url.URL

## method

### EscapedPath

- 功能
    
    负责转义 u.Path

- 注意

    首先判断 u.RawPath 是否是一个有效的转义 path，如果是则直接返回 u.RawPath, 否则会尝试转义
返回转义后的字符串。

    通常建议使用 EscapedPath() 方法而不是直接使用 u.RawPath.