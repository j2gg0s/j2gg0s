# [GoLang] 门外汉系列: Go Assmebly 解读

Go 有一层抽象的汇编层来屏蔽不同平台和机器的差异.

考虑如下代码:
```
     1  package main
     2
     3  //go:noinline
     4  func add(a, b int32) (int32, bool) {
     5          return a + b, true
     6  }
     7
     8  func main() {
     9          add(3, 4)
    10  }
```

通过 go1.17 编译后得到的 Go 汇编代码主体如下:
```
$ GOOS=linux GOARCH=amd64 go tool compile -S main.go

"".add STEXT nosplit size=8 args=0x8 locals=0x0 funcid=0x0
	0x0000 00000 (main.go:4)	TEXT	"".add(SB), NOSPLIT|ABIInternal, $0-8
	0x0000 00000 (main.go:4)	FUNCDATA	$0, gclocals·33cdeccccebe80329f1fdbee7f5874cb(SB)
	0x0000 00000 (main.go:4)	FUNCDATA	$1, gclocals·33cdeccccebe80329f1fdbee7f5874cb(SB)
	0x0000 00000 (main.go:4)	FUNCDATA	$5, "".add.arginfo1(SB)
	0x0000 00000 (main.go:5)	ADDL	BX, AX
	0x0002 00002 (main.go:5)	MOVL	$1, BX
	0x0007 00007 (main.go:5)	RET
	0x0000 01 d8 bb 01 00 00 00 c3                          ........
"".main STEXT size=54 args=0x0 locals=0x10 funcid=0x0
	0x0000 00000 (main.go:8)	TEXT	"".main(SB), ABIInternal, $16-0
	0x0000 00000 (main.go:8)	CMPQ	SP, 16(R14)
	0x0004 00004 (main.go:8)	PCDATA	$0, $-2
	0x0004 00004 (main.go:8)	JLS	47
	0x0006 00006 (main.go:8)	PCDATA	$0, $-1
	0x0006 00006 (main.go:8)	SUBQ	$16, SP
	0x000a 00010 (main.go:8)	MOVQ	BP, 8(SP)
	0x000f 00015 (main.go:8)	LEAQ	8(SP), BP
	0x0014 00020 (main.go:8)	FUNCDATA	$0, gclocals·33cdeccccebe80329f1fdbee7f5874cb(SB)
	0x0014 00020 (main.go:8)	FUNCDATA	$1, gclocals·33cdeccccebe80329f1fdbee7f5874cb(SB)
	0x0014 00020 (main.go:9)	MOVL	$3, AX
	0x0019 00025 (main.go:9)	MOVL	$4, BX
	0x001e 00030 (main.go:9)	PCDATA	$1, $0
	0x001e 00030 (main.go:9)	NOP
	0x0020 00032 (main.go:9)	CALL	"".add(SB)
	0x0025 00037 (main.go:10)	MOVQ	8(SP), BP
	0x002a 00042 (main.go:10)	ADDQ	$16, SP
	0x002e 00046 (main.go:10)	RET
	0x002f 00047 (main.go:10)	NOP
	0x002f 00047 (main.go:8)	PCDATA	$1, $-1
	0x002f 00047 (main.go:8)	PCDATA	$0, $-2
	0x002f 00047 (main.go:8)	CALL	runtime.morestack_noctxt(SB)
	0x0034 00052 (main.go:8)	PCDATA	$0, $-1
	0x0034 00052 (main.go:8)	JMP	0
	0x0000 49 3b 66 10 76 29 48 83 ec 10 48 89 6c 24 08 48  I;f.v)H...H.l$.H
	0x0010 8d 6c 24 08 b8 03 00 00 00 bb 04 00 00 00 66 90  .l$...........f.
	0x0020 e8 00 00 00 00 48 8b 6c 24 08 48 83 c4 10 c3 e8  .....H.l$.H.....
	0x0030 00 00 00 00 eb ca                                ......
	rel 33+4 t=7 "".add+0
	rel 48+4 t=7 runtime.morestack_noctxt+0
go.cuinfo.packagename. SDWARFCUINFO dupok size=0
	0x0000 6d 61 69 6e                                      main
""..inittask SNOPTRDATA size=24
	0x0000 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00  ................
	0x0010 00 00 00 00 00 00 00 00                          ........
gclocals·33cdeccccebe80329f1fdbee7f5874cb SRODATA dupok size=8
	0x0000 01 00 00 00 00 00 00 00                          ........
"".add.arginfo1 SRODATA static dupok size=5
	0x0000 00 04 04 04 ff                                   .....
```

### add
```
	0x0000 00000 (main.go:4)	TEXT	"".add(SB), NOSPLIT|ABIInternal, $0-8
```
- `0x0000 00000`: 相对位置
- `TEXT "".add`: 向 `.text` 中添加新的符号 `add`. 前置的空字符串会在 link 阶段被替换成具体的包名.
- `(SB)`: SB 是 Go Assembly 中的一个虚拟寄存器, static base pointer.
- `NOSPLIT|ABIInternal`: 编译过程中添加的标志位.
- `$0-8`, frame 和 argument 的大小, 0 代表函数的 stack frame 为 0 bytes, 8 代表函数的参数占据 8 bytes.

```
	0x0000 00000 (main.go:4)	FUNCDATA	$0, gclocals·33cdeccccebe80329f1fdbee7f5874cb(SB)
	0x0000 00000 (main.go:4)	FUNCDATA	$1, gclocals·33cdeccccebe80329f1fdbee7f5874cb(SB)
	0x0000 00000 (main.go:4)	FUNCDATA	$5, "".add.arginfo1(SB)
```
FUNCDATA 和后续的 PCDATA 都是编译器引入的为 GC 提供额外信息的指令, 后续不再额外讨论.

```
	0x0000 00000 (main.go:5)	ADDL	BX, AX
	0x0002 00002 (main.go:5)	MOVL	$1, BX
```
按照 Go 的调用约定(call convention), caller 会将 add 的两个参数存放到寄存器 AX, BX.
`ADDL` 将寄存器 AX, BX 相加并将结果存放到 AX.
`MOVL` 将 1(true) 存放到 BX.

```
	0x0007 00007 (main.go:5)	RET
```
`RET` 返回 Caller. 同样的按照调用约定, 返回的两个参数被放到寄存器 AX, BX.

### main
```
	0x0000 00000 (main.go:8)	CMPQ	SP, 16(R14)
	0x0004 00004 (main.go:8)	JLS	47
    ...
	0x002f 00047 (main.go:10)	NOP
	0x002f 00047 (main.go:8)	CALL	runtime.morestack_noctxt(SB)
	0x0034 00052 (main.go:8)	JMP	0
```
SP 是一个虚拟的指针, stack pointer.
寄存 R14 代表 [g structure](https://github.com/golang/go/blob/go1.17.13/src/runtime/runtime2.go#L405),
偏移 14bytes 则对应 stackguard0.
JLS 对应 x86 中的 JBE, jump if below or equal.
整体而言, 这儿是判断 g stack 是否有足够的空间, 如果没有则扩容后再执行后续.

```
	0x0006 00006 (main.go:8)	SUBQ	$16, SP
```
stack 分配 16bytes, 注意因为 stack 从高位往低位扩展, 所以扩容对应 SUB.

```
	0x000a 00010 (main.go:8)	MOVQ	BP, 8(SP)
	0x000f 00015 (main.go:8)	LEAQ	8(SP), BP
	0x0014 00020 (main.go:9)	MOVL	$3, AX
	0x0019 00025 (main.go:9)	MOVL	$4, BX
    ...
	0x0020 00032 (main.go:9)	CALL	"".add(SB)
	0x0025 00037 (main.go:10)	MOVQ	8(SP), BP
```
按照调用约定, 更新 BP, 并将调用参数存放到寄存器 AX, BX.
随后调用函数 add 后, 再恢复原来的 BP.

```
	0x002a 00042 (main.go:10)	ADDQ	$16, SP
```
stack 回收之前分配的 16bytes.

如果对 Go Assembly 感兴趣, 或者困惑与上述不专业的论述, 一些专业和高质量的资料:
- [A Quick Guide to Go's Assembler](https://go.dev/doc/asm)
- [A Primer on Go Assembly](https://github.com/teh-cmc/go-internals/blob/master/chapter1_assembly_primer/README.md)
