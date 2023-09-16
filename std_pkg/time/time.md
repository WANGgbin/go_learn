描述 go 标准库中 time 的实现原理。

实际上，最关键的就两步：

- 通过系统调用从内核读取时间戳
  
  以 linux_amd64 操作系统为例，最终通过调用系统调用 `SYS_clock_gettime` 获取 utc 秒数和纳秒数。
  
- 通过读取系统的配置获取主机当前所在的时区

    以 Ubuntu 为例，时区的配置可以参考 linux_learn/time.md 相关内容。
    
    关于 time 包中如何读取当前时区，可以参考函数 `initLocal`.
  
# time 定义
我们来看看 time 的定义：
```go
type Time struct {
	wall uint64 // 秒/纳秒数，由 Sys_clock_gettime 设置
	ext int64
	
	loc *Location   // 时区信息
}
```

# 跟时区无关函数

- Unix()

  我们都知道通过 Unix() 来获取自 1970-01-01 以来的秒数，跟时区无关，直接通过 Time.wall 返回接口。

- UnixNano()

  同 Unix()，只不过返回纳秒数。

# 跟时区相关函数

- Hour/Minute/Second

  这些时间是跟时区相关的，所以需要基于 Time.wall & Time.loc 来调整时间。
  
  这些函数底层都是通过调用 `Time.abs` 来获取跟时区相关的秒数，我们看看该函数的实现：
  ```go
	// abs returns the time t as an absolute time, adjusted by the zone offset.
	// It is called when computing a presentation property like Month or Hour.
	func (t Time) abs() uint64 {
		l := t.loc
		// Avoid function calls when possible.
		if l == nil || l == &localLoc {
			 // 通过读取操作系统配置文件获取时区信息。
			 l = l.get()
		}
		sec := t.unixSec()
		if l != &utcLoc {
			 if l.cacheZone != nil && l.cacheStart <= sec && sec < l.cacheEnd {
				   sec += int64(l.cacheZone.offset)
			 } else {
				// 在 sec 基础上根据时区信息进行调整
				   _, offset, _, _ := l.lookup(sec)
				   sec += int64(offset)
			 }
		}
		return uint64(sec + (unixToInternal + internalToAbsolute))
	}
  ```

# 几个比较重要的函数

- Date()

  基于 year, month, ..., second, nanosecond, loc 来构造 Time。