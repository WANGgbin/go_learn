package main

import (
	"runtime/debug"
)

// go 发生 panic 的时候，是如何进行栈回溯的呢？ 类似于下面的打印信息：
// goroutine 1 [running]:
// runtime/debug.Stack()
//         /home/wgb/Downloads/go/src/runtime/debug/stack.go:24 +0x65
// runtime/debug.PrintStack()
//         /home/wgb/Downloads/go/src/runtime/debug/stack.go:16 +0x19
// main.a(0xa)
//         /home/wgb/work/learn/golang_learn/stack_trace_learn/stack_trace.go:14 +0x1b
// main.main()
//         /home/wgb/work/learn/golang_learn/stack_trace_learn/stack_trace.go:10 +0x26

// 要打印上述信息，我们需要知道
// 1、发生 panic 语句所才的文件名、行号
// 2、当前 goroutine 的调用栈信息 以及每一个函数的函数名、函数调用所在的文件名、行号、调用函数的参数等

// go 是如何获取这些信息的呢？
// 事实上，在编译的时候， go 会把函数信息以及函数内部每一个 pc(汇编指令) 对应的相关信息(所在文件、行号、所在函数的 sp 相较于 fp（可以理解为： bp） 的偏移量)都编译进 moduledata 中
// 这里函数信息，定义为：

// Layout of in-memory per-function information prepared by linker
// See https://golang.org/s/go12symtab.
// Keep in sync with linker (../cmd/link/internal/ld/pcln.go:/pclntab)
// and with package debug/gosym and with symtab.go in package runtime.
// type _func struct {
// 	entry   uintptr // start pc
// 	nameoff int32   // function name

// 	args        int32  // in/out args size
// 	deferreturn uint32 // offset of start of a deferreturn call instruction from entry, if any.

// 	pcsp      uint32 // sp 相较于 fp 偏移量编码信息在 moduledata.pctab 中偏移量
// 	pcfile    uint32 // entry 所在文件名编码信息在 moduledata.pctab 中便宜量
// 	pcln      uint32 // entry 对应行号编码信息在 moduledata.pctab 中偏移量
// 	npcdata   uint32
// 	cuOffset  uint32 // runtime.cutab offset of this function's CU
// 	funcID    funcID // set for certain special runtime functions
// 	flag      funcFlag
// 	_         [1]byte // pad
// 	nfuncdata uint8   // must be last, must end on a uint32-aligned boundary
// }

// 要获取函数内部某条指令对应的 pcData 信息，就从函数的 entry 开始，一条条指令开始遍历，到 curPC > targetPC的时候，
// prevPC 的 pcData 就是 targetPC 的 pcData 信息。
// 我们重点分析一下 pcValue 函数，函数实现上述功能。（删减一部分不重要的逻辑）

// Returns the PCData value, and the PC where this value starts.
// TODO: the start PC is returned only when cache is nil.
// func pcvalue(f funcInfo, off uint32, targetpc uintptr, cache *pcvalueCache, strict bool) (int32, uintptr) {
// 	datap := f.datap  // datap 就是函数所在的 moduledata
// 	p := datap.pctab[off:]  // pctab 中定义了所有 pc 的相关信息，这里 off 表示 f 的第一条指令的某一类信息（文件、行号）的偏移量
// 	pc := f.entry
// 	prevpc := pc
// 	val := int32(-1)
// 	for {
// 		var ok bool
// 		p, ok = step(p, &pc, &val, pc == f.entry)  // step 获取当前 pc 某个属性到 val 中，同时调整 pc 为下一条指令的地址
// 		if !ok {
// 			break
// 		}
// 		if targetpc < pc {  // 如果 targetpc < pc，则 val 就是 targetpc 某个属性的值

// 			return val, prevpc
// 		}
// 		prevpc = pc
// 	}
// }

// 总而言之，从函数 f 的第一条指令开始，就可以获取到任意一条指令的相关信息

// 接着还需要打印 caller 的栈信息，关键在于定位 调用函数的指令地址，知道该地址后，便可以通过
// 上述方法打印相关信息。那么，如何获取到 caller 调用 callee 的地址呢？

// 实际上，在函数调用栈中，在执行 call 指令的时候，会把下一条地址存储到栈中，我们可以通过当前
// 函数栈的 sp 以及 函数栈帧的长度(偏移量定位在 f.pcsp 中)来计算该指令的值。

// 源码实现
// 1. 首先获取 fp frame.fp = frame.sp + uintptr(funcspdelta(f, frame.pc, &cache))
// 2. 基于 fp 获取存储在栈上的指令地址：
// lrPtr = frame.fp - sys.PtrSize
// frame.lr = uintptr(*(*uintptr)(unsafe.Pointer(lrPtr)))


// 另为一个要注意的问题是函数参数的打印，函数的参数信息（1、相较于 fp 的偏移量 2、参数大小( <= 8, 默认为 8 字节)）也是编码在 moduledata 中的
// 然后依次以 16 进制形式打印，如果参数大小 小于 8 字节，会进行相应调整。可以参考 traceback.go::printArgs(f, argp)实现。
// 所以我们看到 对于复合结构（结构体、map、slice等），打印的都是一个个 8 字节的值。

// 总结：
// 知道文件名、行号、函数名、参数信息是如何获取的，具体可以参考函数： runtime/traceback.go::gentraceback()

func main() {
	s := []int32{}
	a(s)
}

func a(num []int32) {
	debug.PrintStack()
}