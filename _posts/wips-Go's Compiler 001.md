Go 在 Plan9 的基础上定义了自己的汇编语言.
代码经过编译后会生成对应的汇编语言, 随后根据目标平台生成精确的, 机器相关的指令.

具体可以参见 [A Quick Guide to Go's Assembler][]:
> The assembler is based on the input style of the Plan 9 assemblers, which is documented in detail elsewhere. If you plan to write assembly language, you should read that document although much of it is Plan 9-specific. The current document provides a summary of the syntax and the differences with what is explained in that document, and describes the peculiarities that apply when writing assembly code to interact with Go.
> The most important thing to know about Go's assembler is that it is not a direct representation of the underlying machine. Some of the details map precisely to the machine, but some do not. This is because the compiler suite (see this description) needs no assembler pass in the usual pipeline. Instead, the compiler operates on a kind of semi-abstract instruction set, and instruction selection occurs partly after code generation. The assembler works on the semi-abstract form, so when you see an instruction like MOV what the toolchain actually generates for that operation might not be a move instruction at all, perhaps a clear or load. Or it might correspond exactly to the machine instruction with that name. In general, machine-specific operations tend to appear as themselves, while more general concepts like memory move and subroutine call and return are more abstract. The details vary with architecture, and we apologize for the imprecision; the situation is not well-defined.

作为一个编译的门外汉来了解 Go 的编译逻辑的一个问题就是 Plan9 的资料稀缺,
导致理解 Go 的汇编结果时很容易卡在某个点.
所以我开了下脑洞, 既然 x64 的汇编资源非常全, 不如我们先看最终生成的 x64 汇编, 再来看 Go 汇编.


全文使用 go1.21.1, 编译针对 GOOS=linux GOARCH=amd64.
[X64 Cheat Sheet]() 是一份非常好的 X64 汇编入门文档, 可以按需阅读.

编译的命令为 `GOOS=linux GOARCH=amd64 go21 build main.go`,
通过 objdump 获取 x64 汇编结果 ` objdump -j .text -S main > objdump`,
通过 go tool objdump 获取 Go 汇编结果 `go21 tool objdump main > goobjdump`.

## add
```go
//go:noinline
func add(a, b int) (int, bool) {
	return a + b, true
}
```

生成的 x64 汇编:
```
cat -n objdump | grep "<main.add>:" -A 10
129310  0000000000457680 <main.add>:
129311  ;       return a + b, true
129312    457680: 48 01 d8                      addq    %rbx, %rax
129313    457683: bb 01 00 00 00                movl    $1, %ebx
129314    457688: c3                            retq
```

Go 在 1.17 从 stack-based calling convention 切换到了 register-based calling convention,
即之前通过 stack 在调用函数时传递参数和返回值, 这是 Plan9 的惯例,
之后通过寄存器传递参数和返回值, 带来了性能的提升.

但在寄存器的使用上, Go 应该没有遵循 x64 的默认规则.
调用者(caller)将 add 的两个参数存放在寄存器 rax 和 rbx.
被调用者(callee)将两个返回值存放在寄存器 rax 和 ebx.

- `addq %rbx, %rax` 将寄存器 rbx 中的值和 rax 中的值相加后保存到寄存器 rax.
- `movl $1, %ebx` 将 true 存放到寄存器 ebx.

## Reference
- [Introduction to the Go Compiler](https://github.com/golang/go/tree/go1.17.13/src/cmd/compile)
- [X64 Cheat Sheet](https://cs.brown.edu/courses/cs033/docs/guides/x64_cheatsheet.pdf)
- [Introduction to the Go compiler's SSA backend](https://github.com/golang/go/blob/go1.17.13/src/cmd/compile/internal/ssa/README.md)
- [A Quick Guide to Go's Assembler](https://go.dev/doc/asm)
