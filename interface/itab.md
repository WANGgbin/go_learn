描述 go 中 itab 相关的操作.

go 如果判断某个类型是否实现某个接口呢? 假设接口类型为 i, 动态类型为 t, 实际上,在给接口赋值的时候, 会依次遍历 i 对应的 imethod, 对每一个 imethod, 会查找 t 对应的 methods(二分查找), 然后判断 method 对应的 func 类型的输入/输出参数个数以及类型是否跟 imethod 对应的 func 类型一致.如果全部一致,则认为类型 t 实现了接口. 详细实现可以参考:
```go
func (m *itab) init() string {
	inter := m.inter
	typ := m._type
	x := typ.uncommon()

	// both inter and typ have method sorted by name,
	// and interface names are unique,
	// so can iterate over both in lock step;
	// the loop is O(ni+nt) not O(ni*nt).
	ni := len(inter.mhdr)
	nt := int(x.mcount)
	xmhdr := (*[1 << 16]method)(add(unsafe.Pointer(x), uintptr(x.moff)))[:nt:nt]
	j := 0
	methods := (*[1 << 16]unsafe.Pointer)(unsafe.Pointer(&m.fun[0]))[:ni:ni]
	var fun0 unsafe.Pointer
imethods:
    // 遍历每一个 imethod
	for k := 0; k < ni; k++ {
		i := &inter.mhdr[k]
		itype := inter.typ.typeOff(i.ityp)
		name := inter.typ.nameOff(i.name)
        // 获取函数名以及 pkg 路径
		iname := name.name()
		ipkg := name.pkgPath()
		if ipkg == "" {
			ipkg = inter.pkgpath.name()
		}
        // 依次遍历 typ 中的函数, 因为 typ.method 是根据函数名排序的,所以从上次 j 的位置继续判断即可.
		for ; j < nt; j++ {
			t := &xmhdr[j]
			tname := typ.nameOff(t.name)
            // 只有类型和名字一致才继续判断. 由此可知,自定义类型的函数类别,并没有包含 reciver!!!
			if typ.typeOff(t.mtyp) == itype && tname.name() == iname {
				pkgPath := tname.pkgPath()
				if pkgPath == "" {
					pkgPath = typ.nameOff(x.pkgpath).name()
				}
                // 如果自定义类型的函数是导出的 或者 (不导出但是接口和方法在同一个pkg)!!!
				if tname.isExported() || pkgPath == ipkg {
					if m != nil {
                        // 根据 ifn 获得函数的地址
						ifn := typ.textOff(t.ifn)
						if k == 0 {
							fun0 = ifn // we'll set m.fun[0] at the end
						} else {
                            // 赋值到 methods 中
							methods[k] = ifn
						}
					}
					continue imethods
				}
			}
		}
		// didn't find method
		m.fun[0] = 0
		return iname
	}
	m.fun[0] = uintptr(fun0)
	return ""
}
```
上述函数实际上是初始化 itab 的过程,里面夹杂了某个类型是否实现某个接口的判断逻辑. 但是如果每次都需要这么判断的话,会影响到程序的性能.实际上,当接口类型和动态类型确定的时候,动态类型是否实现接口类型的结果是确定的.因此,为了性能考虑,可以将接口类型和动态类型的关系缓存下来,下次直接通过缓存的信息判断即可.而这就是 itab 存在的意义.

## itab
itab 的定义如下:
```go
type itab struct {
    inter *interfacetype  // 接口类型
    _type *_type  // 实际类型
    hash  uint32 // copy of _type.hash. Used for type switches.
    _     [4]byte
    fun   [1]uintptr // variable sized. fun[0]==0 means _type does not implement inter. 如果有多个 func,紧跟在 fun 之后
}
```
每当判断接口 i 内部的类型是不是 t 的时候,通过 i + t 从全局 itabTable 中查找对应的 itab. 如果 itab 存在且 itab.func[0] != 0, 则认为 i 的动态类型就是 t. 如果 itab.func[0] == 0, 则认为 _type 没有实现 inter, 如果 itab 不存在则需要调用前面的 init() 函数来初始化 itab 并将 itab 加入到全局 itab 缓存中.

## itabTable
itabTable 是 itab 的全局缓存. 实际上是个哈系表, 通过线性探测法来解决冲突.

实际上 itab 在整个进程的生命间内都有效,不应该涉及 gc. 那如何分配 itab 结构体呢? 实际上是通过 `persistentalloc` 来完成内存分配的. go 进程内部维护了一个 `persistentChunks`, 改变量实际上就是 persistentChunk 的链表, 每个 persistentChunk 都是通过 `sysAlloc` 在进程虚拟空间分配的一块儿连续内存,整个链表通过 chunk 的首个 8 字节链在一起. 我们来看看 `persistentalloc`的是下.
```go
// 描述每一个 persistentAlloc, 注意 persistent 分配的内存没有 free 操作
type persistentAlloc struct {
    // 基地址
	base *notInHeap
	// 分配到哪儿了
    off  uintptr
}
// 会从当前 p 的 persistentAlloc 分配 size 大小空间,如果不够则重新分配一个 persistentAlloc
func persistentalloc1(size, align uintptr, sysStat *sysMemStat) *notInHeap {
	const (
		maxBlock = 64 << 10 // VM reservation granularity is 64K on windows
	)

    align = 8

	mp := acquirem()
	var persistent *persistentAlloc
    // 每个 p 维护了一个 persistentAlloc
	if mp != nil && mp.p != 0 {
		persistent = &mp.p.ptr().palloc
	} else {
		lock(&globalAlloc.mutex)
		persistent = &globalAlloc.persistentAlloc
	}
	persistent.off = alignUp(persistent.off, align)
    // 如果当前的 persistentAlloc 不足以容纳 size 大小 或者 当前 p 还没初始化 persistentAlloc, 则调用 sysAlloc 分配一个 persistentAlloc
    // persistentChunkSize 为每一个 persistentAlloc 的大小
	if persistent.off+size > persistentChunkSize || persistent.base == nil {
		persistent.base = (*notInHeap)(sysAlloc(persistentChunkSize, &memstats.other_sys))

        // 加入到全局列表中
		for {
			chunks := uintptr(unsafe.Pointer(persistentChunks))
			*(*uintptr)(unsafe.Pointer(persistent.base)) = chunks
            // 全局列表由 persistentChunks 定义, 因为跟 gc 没关系,又因为 persistentChunks 的初始值为 0, 所以 persistentChunks 定义在 .noptrbss 分节中.
            // 可以通过 `objdump -j .noptrbss -t ./a.out` 查看 symbol: persistentChunks 的信息
            // 00000000004f1e10 g     O .noptrbss      0000000000000008 runtime.persistentChunks
			if atomic.Casuintptr((*uintptr)(unsafe.Pointer(&persistentChunks)), chunks, uintptr(unsafe.Pointer(persistent.base))) {
				break
			}
		}
		persistent.off = alignUp(sys.PtrSize, align)
	}
    // 获取地址 p, 并调整 persistent.off
	p := persistent.base.add(persistent.off)
	persistent.off += size
	releasem(mp)
	if persistent == &globalAlloc.persistentAlloc {
		unlock(&globalAlloc.mutex)
	}
	return p
}
``` 
以上便是 go 内部 persistentAlloc 的实现, 分配的内存永久有效没有 free 操作, gc 感知不到这些内存信息(通过将 persistentChunks 置于 .noptrbss 实现).

itabTable 是支持扩容的, 这就有个问题, 既然 itab 对应的内存是 persistent, 那么 rehash 后,旧的哈系表的 bucket 怎么释放呢? 实际上,虽然 itab 是 persistent 的, 但是 itabTable 还是在 data 分节中的,
这样当扩容后, itabTable 指向新的 buckets ,旧的 buckets 就会被 gc 回收了.
```text
00000000004c0350 g     O .data  0000000000000008 runtime.itabTable
```

### 哈系表的查找和添加
关于 itabTable 值得我们仔细学习的是 itabTable 的操作. 参考: `runtime/iface.go`. 为了保证最大的效率, 哈系表的读操作是不加锁的,只有写操作才会加锁. itabTable 的操作是个读多写少的场景.
当添加元素需要扩容的时候, 先创建一个新的 table, 然后 **原子性**的设置 itabTable, 使得后续的读操作可以直接新的 itabTable.

这里有个问题, 如果 itab 正在扩容中, find 并没有在 itabTable 中发现 itab, 下一步该如何操作呢? go 给的解决方案是加锁, 线程睡眠, 等到获取锁的时候, 此时 itabTable 指向的就是最新的 table, 则再次
执行 find 即可.

即使不扩容,某个线程正在添加某个 itab(原子性设置, 保证设置后, 能够被 find 及时发现, 使用了**原子操作全局可见的特性**.), 但是发生在 find 之后, find 也找不到 itab, 此时同样需要加锁, 恢复后,再次查找 find 即可.