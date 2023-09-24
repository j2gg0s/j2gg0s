Go 在 Plan9 的基础上定义了自己的汇编语言.
代码经过编译后会生成对应的汇编语言, 随后根据目标平台生成精确的, 机器相关的指令.

具体可以参见 [A Quick Guide to Go's Assembler][]:
> The assembler is based on the input style of the Plan 9 assemblers, which is documented in detail elsewhere. If you plan to write assembly language, you should read that document although much of it is Plan 9-specific. The current document provides a summary of the syntax and the differences with what is explained in that document, and describes the peculiarities that apply when writing assembly code to interact with Go.

作为一个编译的门外汉来了解 Go 的编译逻辑的一个问题就是 Plan9 的资料稀缺,
导致理解 Go 的汇编结果时很容易卡在某个点.
所以我开了下脑洞, 既然 x64 的汇编资源非常全, 不如我们先看最终生成的 x64 汇编, 再来看 Go 汇编.
[X64 Cheat Sheet][] 是一份非常好的 X64 汇编入门文档, 可以按需阅读.

全文使用 go1.21.1, 编译针对 linux/amd64.
- 编译的命令为 `GOOS=linux GOARCH=amd64 go21 build main.go`,
- 通过 objdump 获取 x64 汇编结果 `objdump -j .text -S main > objdump`,
- 通过 go tool objdump 获取 Go 汇编结果 `go21 tool objdump main > goobjdump`.

## add
```go
//go:noinline
func add(a, b int) (int, bool) {
	return a + b, true
}
```
注解 `//go:noinline` 用于告诉编译器不要进行 inline 优化,
即避免编译器自动将调用这些函数的地方替换成函数代码.

Go 在 1.17 从 stack-based calling convention 切换到了 register-based calling convention,
即之前通过 stack 在调用函数时传递参数和返回值, 这是 Plan9 的惯例,
之后通过寄存器传递参数和返回值, 带来了性能的提升.

但在寄存器的使用上, Go 没有遵循 x64 的默认习俗.
调用者(caller) 将 add 的两个参数存放在寄存器 rax 和 rbx.
被调用者(callee) 将两个返回值存放在寄存器 rax 和 rbx.

生成的 x64 汇编:
```
cat -n objdump | grep "<main.add>:" -A 10
129310  0000000000457680 <main.add>:
129311  ;       return a + b, true
129312    457680: 48 01 d8                      addq    %rbx, %rax      ; 调用参数被保存在寄存器 rax 和 rbx
129313    457683: bb 01 00 00 00                movl    $1, %ebx        ; ebx 和 rbx 是同一个寄存器, ebx 对应前 4 字节, rbx 对应全部的 8 字节
129314    457688: c3                            retq                    ; 返回值已经保存在寄存器 rax 和 rbx 内
```

对应的 Go 汇编:
```
cat -n goobjdump | grep "TEXT main.add" -A 10
 82843  TEXT main.add(SB) /Users/j2gg0s/go/src/github.com/j2gg0s/j2gg0s/examples/ssa/main.go
 82844    main.go:5             0x457680                4801d8                  ADDQ BX, AX
 82845    main.go:5             0x457683                bb01000000              MOVL $0x1, BX
 82846    main.go:5             0x457688                c3                      RET
```

## if
```go
//go:noinline
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
```

生成的 x86 汇编:
```
cat -n objdump | grep "<main.max>:" -A 30
129339  00000000004576a0 <main.max>:
129340  ;       if a > b {
129341    4576a0: 48 39 c3                      cmpq    %rax, %rbx              ; cmp 将 rbx-rax 的结果保存到条件寄存器
129342    4576a3: 7d 01                         jge     0x4576a6 <main.max+0x6> ; 如果 cmp 的结果大于等于 0, 则跳转到对应地址的指令
129343  ;               return a
129344    4576a5: c3                            retq
129345  ;       return b
129346    4576a6: 48 89 d8                      movq    %rbx, %rax              ; 返回结果需要保存到 rax, 所以需要将 rbx 的值转移到 rax
129347    4576a9: c3                            retq
```

对应的 Go 汇编:
```
cat -n goobjdump | grep "TEXT main.max" -A 10
 82848  TEXT main.max(SB) /Users/j2gg0s/go/src/github.com/j2gg0s/j2gg0s/examples/ssa/main.go
 82849    main.go:10            0x4576a0                4839c3                  CMPQ BX, AX
 82850    main.go:10            0x4576a3                7d01                    JGE 0x4576a6
 82851    main.go:11            0x4576a5                c3                      RET
 82852    main.go:13            0x4576a6                4889d8                  MOVQ BX, AX
 82853    main.go:13            0x4576a9                c3                      RET
```

## for
```go
//go:noinline
func sum(numbers []int) int {
	sum := 0
	for i := 0; i < len(numbers); i++ {
		sum += numbers[i]
	}
	return sum
}
```

生成的 x86 汇编:
```
cat -n objdump | grep "<main.sum>:" -A 30
129371  00000000004576c0 <main.sum>:
129372  ; func sum(numbers []int) int {
129373    4576c0: 48 89 44 24 08                movq    %rax, 8(%rsp)       ; 将 rax 的值存放到 stack, TODO: 为什么需要这么做, 为什么是 8.
129374    4576c5: 31 c9                         xorl    %ecx, %ecx          ; 清空寄存器 rcx 的前 4 字节
129375    4576c7: 31 d2                         xorl    %edx, %edx
129376  ;       for i := 0; i < len(numbers); i++ {
129377    4576c9: eb 0a                         jmp     0x4576d5 <main.sum+0x15>
129378  ;               sum += numbers[i]
129379    4576cb: 48 8b 34 c8                   movq    (%rax,%rcx,8), %rsi ; 将 numbers[i] 存在到 rsi, rax 是数组地址, rcx 是 i, 8 代表元素占 8 字节
129380  ;       for i := 0; i < len(numbers); i++ {
129381    4576cf: 48 ff c1                      incq    %rcx
129382  ;               sum += numbers[i]
129383    4576d2: 48 01 f2                      addq    %rsi, %rdx
129384  ;       for i := 0; i < len(numbers); i++ {
129385    4576d5: 48 39 cb                      cmpq    %rcx, %rbx          ; 判断 i < len(numbers)
129386    4576d8: 7f f1                         jg      0x4576cb <main.sum+0xb>
129387  ;       return sum
129388    4576da: 48 89 d0                      movq    %rdx, %rax          ; 返回结果需要存储在 rax
129389    4576dd: c3                            retq
```

对应的 Go 汇编
```
cat -n goobjdump | grep "TEXT main.sum" -A 10
 82855  TEXT main.sum(SB) /Users/j2gg0s/go/src/github.com/j2gg0s/j2gg0s/examples/ssa/main.go
 82856    main.go:17            0x4576c0                4889442408              MOVQ AX, 0x8(SP)
 82857    main.go:17            0x4576c5                31c9                    XORL CX, CX
 82858    main.go:17            0x4576c7                31d2                    XORL DX, DX
 82859    main.go:19            0x4576c9                eb0a                    JMP 0x4576d5
 82860    main.go:20            0x4576cb                488b34c8                MOVQ 0(AX)(CX*8), SI
 82861    main.go:19            0x4576cf                48ffc1                  INCQ CX
 82862    main.go:20            0x4576d2                4801f2                  ADDQ SI, DX
 82863    main.go:19            0x4576d5                4839cb                  CMPQ BX, CX
 82864    main.go:19            0x4576d8                7ff1                    JG 0x4576cb
 82865    main.go:22            0x4576da                4889d0                  MOVQ DX, AX
```

## main
```go
func main() {
	add(10, 20)
	max(10, 20)
	sum([]int{10, 20})
}
```

生成的 x86 汇编:
```
cat -n objdump | grep "<main.main>:" -A 60
129393  00000000004576e0 <main.main>:
129394  ; func main() {
129395    4576e0: 49 3b 66 10                   cmpq    16(%r14), %rsp  ; Go 用于判断 stack 是否需要扩容的方法
129396    4576e4: 76 56                         jbe     0x45773c <main.main+0x5c>
129397    4576e6: 55                            pushq   %rbp            ; 在函数执行之前, 需要将 base pointer 暂存在 stack
129398    4576e7: 48 89 e5                      movq    %rsp, %rbp      ; 并将 stack pointer 的值赋给 bp
129399    4576ea: 48 83 ec 28                   subq    $40, %rsp       ; 在 stack 中预先分配 40 字节
129400  ;       add(10, 20)
129401    4576ee: b8 0a 00 00 00                movl    $10, %eax
129402    4576f3: bb 14 00 00 00                movl    $20, %ebx
129403    4576f8: e8 83 ff ff ff                callq   0x457680 <main.add>
129404  ;       max(10, 20)
129405    4576fd: b8 0a 00 00 00                movl    $10, %eax
129406    457702: bb 14 00 00 00                movl    $20, %ebx
129407    457707: e8 94 ff ff ff                callq   0x4576a0 <main.max>
129408  ;       sum([]int{10, 20})
129409    45770c: 44 0f 11 7c 24 18             movups  %xmm15, 24(%rsp)    ; xmm15 是 16 字节的寄存器, 配合 movups 用于清空栈中 24~40 字节
129410    457712: 48 c7 44 24 18 0a 00 00 00    movq    $10, 24(%rsp)
129411    45771b: 48 c7 44 24 20 14 00 00 00    movq    $20, 32(%rsp)
129412    457724: 48 8d 44 24 18                leaq    24(%rsp), %rax      ; 将数组的地址存放到寄存器 rax
129413    457729: bb 02 00 00 00                movl    $2, %ebx            ; 将数据的元素个数存放到 rbx
129414    45772e: 48 89 d9                      movq    %rbx, %rcx
129415    457731: e8 8a ff ff ff                callq   0x4576c0 <main.sum>
129416  ; }
129417    457736: 48 83 c4 28                   addq    $40, %rsp       ; 释放在函数最初分配给栈的 40 字节
129418    45773a: 5d                            popq    %rbp            ; 恢复 base pointer
129419    45773b: c3                            retq
129420  ; func main() {
129421    45773c: 0f 1f 40 00                   nopl    (%rax)
129422    457740: e8 9b ce ff ff                callq   0x4545e0 <runtime.morestack_noctxt.abi0>
129423    457745: eb 99                         jmp     0x4576e0 <main.main>
```

细心的同学可能会思考为什么调用 main.sum 之前初始化数组是用 24(%rsp). 这是因为虽然 go's calling convention 已经从
stack-base 切换到了 regisger-base. 但是可能出于兼容或者上面目的, 依然在 stack 为通过 register 传递的参数和返回值保留空间.
Ref [Function call argument and result passing](https://go.googlesource.com/go/+/refs/heads/dev.regabi/src/cmd/compile/internal-abi.md#function-call-argument-and-result-passing):
< Beyond the arguments and results passed on the stack, the caller also reserves spill space on the stack for all register-based arguments (but does not populate this space).

对应的 Go 汇编:
```
cat -n goobjdump | grep "TEXT main.main" -A 1000
 82868  TEXT main.main(SB) /Users/j2gg0s/go/src/github.com/j2gg0s/j2gg0s/examples/goasm/main.go
 82869    main.go:25            0x4576e0                493b6610                CMPQ SP, 0x10(R14)
 82870    main.go:25            0x4576e4                7656                    JBE 0x45773c
 82871    main.go:25            0x4576e6                55                      PUSHQ BP
 82872    main.go:25            0x4576e7                4889e5                  MOVQ SP, BP
 82873    main.go:25            0x4576ea                4883ec28                SUBQ $0x28, SP
 82874    main.go:26            0x4576ee                b80a000000              MOVL $0xa, AX
 82875    main.go:26            0x4576f3                bb14000000              MOVL $0x14, BX
 82876    main.go:26            0x4576f8                e883ffffff              CALL main.add(SB)
 82877    main.go:27            0x4576fd                b80a000000              MOVL $0xa, AX
 82878    main.go:27            0x457702                bb14000000              MOVL $0x14, BX
 82879    main.go:27            0x457707                e894ffffff              CALL main.max(SB)
 82880    main.go:28            0x45770c                440f117c2418            MOVUPS X15, 0x18(SP)
 82881    main.go:28            0x457712                48c74424180a000000      MOVQ $0xa, 0x18(SP)
 82882    main.go:28            0x45771b                48c744242014000000      MOVQ $0x14, 0x20(SP)
 82883    main.go:28            0x457724                488d442418              LEAQ 0x18(SP), AX
 82884    main.go:28            0x457729                bb02000000              MOVL $0x2, BX
 82885    main.go:28            0x45772e                4889d9                  MOVQ BX, CX
 82886    main.go:28            0x457731                e88affffff              CALL main.sum(SB)
 82887    main.go:29            0x457736                4883c428                ADDQ $0x28, SP
 82888    main.go:29            0x45773a                5d                      POPQ BP
 82889    main.go:29            0x45773b                c3                      RET
 82890    main.go:25            0x45773c                0f1f4000                NOPL 0(AX)
 82891    main.go:25            0x457740                e89bceffff              CALL runtime.morestack_noctxt.abi0(SB)
 82892    main.go:25            0x457745                eb99                    JMP main.main(SB)
```

## Reference
[Introduction to the Go Compiler]: https://github.com/golang/go/tree/go1.17.13/src/cmd/compile
[X64 Cheat Sheet]: https://cs.brown.edu/courses/cs033/docs/guides/x64_cheatsheet.pdf
[Go internal ABI specification]: https://go.googlesource.com/go/+/refs/heads/dev.regabi/src/cmd/compile/internal-abi.md
