TEXT ·refLocalVar(SB), $4096-8
    MOVQ $0, local-4096(SP)
    LEAQ local-4096(SP), AX
    MOVQ AX, ptr+0(FP)
    RET
