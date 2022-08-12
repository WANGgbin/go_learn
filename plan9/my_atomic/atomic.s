#include "textflag.h" // 因为函数使用了 NOSPLIT 这样的标志,所以必须包含此头文件,否则报: illegal or missing addressing mode for symbol NOSPLIT

TEXT ·AddInt64(SB), NOSPLIT, $0-24
    MOVQ delta+8(FP), AX
    MOVQ ptr+0(FP), BX
    LOCK  // 锁定系统总线,在指令执行期间,禁止其他 cpu 执行
    XADDQ AX, 0(BX)
    ADDQ delta+8(FP), AX
    MOVQ AX, new+16(FP)
    RET

TEXT ·SwapInt64(SB), NOSPLIT, $0-24
    MOVQ addr+0(FP), AX
    MOVQ new+8(FP), BX
    LOCK
    XCHGQ BX, 0(AX)
    MOVQ BX, old+16(FP)
    RET

TEXT ·CompareAndSwapInt64(SB), NOSPLIT, $0-25 // 是 25 而不是 32, 不存在字节对齐???
    MOVQ addr+0(FP), BX
    MOVQ old+8(FP), AX  // 注意,这里只能是 AX, 因为 CMPXCHG 是跟 AX 内容比较的
    MOVQ new+16(FP), CX
    LOCK
    // CMPXCHGQ new+16(FP), 0(BX)  // 两个操作数不能都为 mem
    CMPXCHGQ CX, 0(BX)
    SETEQ swapped+24(FP)
    RET

//最后的空行是必须的,否则会报: unexpected EOF