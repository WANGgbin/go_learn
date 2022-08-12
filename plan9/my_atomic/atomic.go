package my_atomic

func AddInt64(old *int64, delta int64) int64

func SwapInt64(addr *int64, new int64) (old int64)

func CompareAndSwapInt64(addr *int64, old, new int64) (swapped bool)