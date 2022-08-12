TEXT ·Add(SB), $16-24
    MOVQ x+0(FP), AX
    MOVQ AX, local1-8(SP)
    MOVQ y+8(FP), AX
    MOVQ AX, local2-2048(SP)
    MOVQ local1-8(SP), AX
    ADDQ local2-2048(SP), AX
    MOVQ AX, ret+16(FP)
    RET


// TEXT ·Add(SB), $0-24
//     MOVQ x+0(FP), AX
//     ADDQ y+8(FP), AX
//     MOVQ AX, ret+16(FP)
//     RET
