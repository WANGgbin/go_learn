package main

/**
 为什么需要 sync.Pool? 当 app 需要频繁的创建某种类型的对象的时候,可能会触发 app 频繁 的 gc,
 因此,要解决此问题,就需要一种方法,就频繁分配的内存大小控制到一定范围内,从而避免频繁触发 gc. 一种解决方案是提供内存池,在分配对象的时候,通过服用内存的方式,从而比买你内存的无限扩大.
 那么要服用内存的话,该如何知道某一块对象对应的内存是否空闲呢? 要通过 gc 吗? 那不是又回到起点了?

 猜测:
	内存池对象需要用户主动释放,这样 go 才知道那些内存是空闲的, 进而能够复用内存.
	这种方式有个缺点,用户主动释放对象的这种行为与 go 的 gc 是相悖的, 一会儿不需关系对象内存的回收一会儿又要主动回收对象,
	会让用户产生疑惑.我们可以通过提供清晰接口的方式,尽量降低这种方案带来的困惑

 go 实现
1. sync.Pool 的实现原理?
	参考: https://zhuanlan.zhihu.com/p/399150710
	go 标准库要实现对象的复用,需要用户主动把对象扔到 pool 中,这样在下次 Get 的时候,就可以获取到之前的对象.如何使用完对象并没有 Put 的话,
	那么在下次 Get 的时候就只能重新分配内存创建对象, pool 就失去了存在的意义.
	那么 pool 是如何 gc 交互的呢? pool 中有一个全局变量 allPools []*Pool, 通过此全局变量,就可以把 pool 中所有的对象串起来,
	这样 gc 扫描的时候,并不会回收这些对象.
	
2. 怎么使用 sync.Pool 需要注意什么?
	1.不使用的对象,需要 Put 到 pool 中,才可以发挥 pool 的作用. 当然不 put 也没什么问题, 对象会被 gc 回收.
	2. New 需要返回指向对象的指针.

	为了尽量降低竞争, pool 内部会给每一个 P 分配一个 localCache, 在 Get/Put 对象的时候,是操作当前 P 对应的
	数据成员的,所以在这个阶段需要避免 goroutine 抢占调度.否则等到当前 goroutine 下次调度的时候,会造成 localCache
	数据的混乱. 那么禁止被抢占是如何实现的呢?

*/

import (
	"fmt"
	"sync"
)

type Ojbect struct {
	Name string
}

func makeObjectPool() sync.Pool {
	return sync.Pool{
		New: func() interface{} {
			return new(Ojbect)
		},
	}
}

func main() {
	objPool := makeObjectPool()
	obj1 := objPool.Get().(*Ojbect)
	fmt.Printf("%s\n", obj1.Name)
	obj1.Name = "obj1"
	objPool.Put(obj1)
	obj2 := objPool.Get().(*Ojbect)
	fmt.Printf("%s\n", obj2.Name)

}
