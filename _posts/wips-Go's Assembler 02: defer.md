全文依然基于 go1.21.1, GOOS=linux, GOARCH=amd64.

从 [Proposal: Low-cost defers through inline code, and extra funcdata to manage the panic case][] 来看, defer 在 Go 中的实现方式有三种:
- open coded, 即在汇编中, 直接将 defer 插入对应的退出/返回的位置, 性能上和直接调用基本没有区别
- stack-allocated, defer 被保存在栈, 编译时在返回处插入对 runtime.deferreturn 的调用, 有轻微的性能损失
- heap-allocated, 与 stack-allocated 的区别在于, defer 被保存在推, 导致有明显的调用成本

## open coded
Go's Proposal 中的例子解释了大概的实现.
假设代码如下:
```
defer f1(a)
if cond {
 defer f2(b)
}
body...
```

经过编译后的代码如下:
```
deferBits |= 1<<0
tmpF1 = f1
tmpA = a
if cond {
 deferBits |= 1<<1
 tmpF2 = f2
 tmpB = b
}
body...
exit:
if deferBits & 1<<1 != 0 {
 deferBits &^= 1<<1
 tmpF2(tmpB)
}
if deferBits & 1<<0 != 0 {
 deferBits &^= 1<<0
 tmpF1(tmpA)
}
```

即:
- 将 defer 涉及的函数和变量都保存到栈上
- 用 deferBits 来保存对应的 defer 是否应该执行
- 在退出时插入调用代码


```go
//go:noinline
func max(a, b int) int {
        if a > b {
                defer func() {
                        fmt.Println("max is a")
                }()
                return a
        }
        defer func() {
                fmt.Println("max is b")
        }()
        return b
}
```

对应的汇编代码,
- 栈上的 0~6th 字节未使用, 7th 字节被用来存放 deferbits
- 8~15th 字节用于在调用 defer 前暂存 main.max 的返回值
- 16~23th 和 24~31th 两个 quadword 用于引用两个 defer 函数, 可能是避免 gc, 汇编的具体做法没有理解
```
000000000047ae00 <main.max>:
; func max(a, b int) int {
  47ae00: 49 3b 66 10                  	cmpq	16(%r14), %rsp
  47ae04: 0f 86 87 00 00 00            	jbe	0x47ae91 <main.max+0x91>
  47ae0a: 55                           	pushq	%rbp
  47ae0b: 48 89 e5                     	movq	%rsp, %rbp
  47ae0e: 48 83 ec 20                  	subq	$32, %rsp
  47ae12: 44 0f 11 7c 24 10            	movups	%xmm15, 16(%rsp)
  47ae18: c6 44 24 07 00               	movb	$0, 7(%rsp)
  47ae1d: 48 c7 44 24 08 00 00 00 00   	movq	$0, 8(%rsp)
; 	if a > b {
  47ae26: 48 39 d8                     	cmpq	%rbx, %rax              ; rax - rbx
  47ae29: 7e 2b                        	jle	0x47ae56 <main.max+0x56>    ; jle -> jump if less or equal
; 		defer func() {
  47ae2b: 48 8d 0d c6 06 02 00         	leaq	132806(%rip), %rcx      # 0x49b4f8 <go:func.*+0x220>    ; TODO
  47ae32: 48 89 4c 24 18               	movq	%rcx, 24(%rsp)          ; defer 函数被保存在栈中
  47ae37: c6 44 24 07 01               	movb	$1, 7(%rsp)             ; deferbits 的第一个 bit 被置为 1, movb 仅移动一个字节
; 		return a
  47ae3c: 48 89 44 24 08               	movq	%rax, 8(%rsp)           ; 调用 defer 将 rax 中的返回结果暂存到栈中
  47ae41: c6 44 24 07 00               	movb	$0, 7(%rsp)             ; 清空 deferbits 的第一个 bit
  47ae46: e8 b5 00 00 00               	callq	0x47af00 <main.max.func1>
  47ae4b: 48 8b 44 24 08               	movq	8(%rsp), %rax
  47ae50: 48 83 c4 20                  	addq	$32, %rsp
  47ae54: 5d                           	popq	%rbp
  47ae55: c3                           	retq
; 	defer func() {
  47ae56: 48 8d 05 a3 06 02 00         	leaq	132771(%rip), %rax      # 0x49b500 <go:func.*+0x228>    ; TODO
  47ae5d: 48 89 44 24 10               	movq	%rax, 16(%rsp)
  47ae62: c6 44 24 07 02               	movb	$2, 7(%rsp)             ; 第二个 defer 对应 deferbits 的第二个 bit
; 	return b
  47ae67: 48 89 5c 24 08               	movq	%rbx, 8(%rsp)
  47ae6c: c6 44 24 07 00               	movb	$0, 7(%rsp)
  47ae71: e8 ea 00 00 00               	callq	0x47af60 <main.max.func2>
  47ae76: 48 8b 44 24 08               	movq	8(%rsp), %rax
  47ae7b: 48 83 c4 20                  	addq	$32, %rsp
  47ae7f: 5d                           	popq	%rbp
  47ae80: c3                           	retq
  47ae81: e8 5a 47 fb ff               	callq	0x42f5e0 <runtime.deferreturn>  ; retq 之后的这部分配合 panic, 具体后续展开
  47ae86: 48 8b 44 24 08               	movq	8(%rsp), %rax
  47ae8b: 48 83 c4 20                  	addq	$32, %rsp
  47ae8f: 5d                           	popq	%rbp
  47ae90: c3                           	retq
; func max(a, b int) int {
  47ae91: 48 89 44 24 08               	movq	%rax, 8(%rsp)
  47ae96: 48 89 5c 24 10               	movq	%rbx, 16(%rsp)
  47ae9b: 0f 1f 44 00 00               	nopl	(%rax,%rax)
  47aea0: e8 fb fb fd ff               	callq	0x45aaa0 <runtime.morestack_noctxt.abi0>
  47aea5: 48 8b 44 24 08               	movq	8(%rsp), %rax
  47aeaa: 48 8b 5c 24 10               	movq	16(%rsp), %rbx
  47aeaf: e9 4c ff ff ff               	jmp	0x47ae00 <main.max>
```

## stack-allocated
当 defer 的数量较多, 比如超过 8 个, 为了避免生成的汇编代码体积膨胀等问题, Go 会选择使用 stack/heap allocated 的方式实现 defer.

stack allocated 的模式下:
- 在遇到 defer 时, 通过调用 runtime.deferprocStack 以链表的形式将 defer 保存到当前 goroutine
- 在返回时, 插入对 runtime.deferreturn 的的调用, 按 FILO 的顺序执行当前 goroutine 的 defer 链表

Go 示例代码:
```go
//go:noinline
func add(a, b int) int {
	defer func() { fmt.Println(1) }()
	defer func() { fmt.Println(2) }()
	defer func() { fmt.Println(3) }()
	defer func() { fmt.Println(4) }()
	defer func() { fmt.Println(5) }()
	defer func() { fmt.Println(6) }()
	defer func() { fmt.Println(7) }()
	defer func() { fmt.Println(8) }()
	defer func() { fmt.Println(9) }()
	return a + b
}
```

通过 deferprocStack 将 defer 保存到 goroutine 的汇编如下.
```
; 	defer func() { fmt.Println(1) }()
  47ae56: 48 8d 0d 8b 16 02 00         	leaq	136843(%rip), %rcx      # 0x49c4e8 <go:func.*+0x220>
  47ae5d: 48 89 8c 24 d8 01 00 00      	movq	%rcx, 472(%rsp)
  47ae65: 48 8d 84 24 c0 01 00 00      	leaq	448(%rsp), %rax
  47ae6d: e8 8e 41 fb ff               	callq	0x42f000 <runtime.deferprocStack>
  47ae72: 85 c0                        	testl	%eax, %eax      ; TODO
  47ae74: 0f 85 d9 01 00 00            	jne	0x47b053 <main.add+0x233>
```

理解上述汇编, 需要结合 deferprocStack 的函数签名.
```go
func deferprocStack(d *_defer) {}

type _defer struct {
	started bool
	heap    bool
	// openDefer indicates that this _defer is for a frame with open-coded
	// defers. We have only one defer record for the entire frame (which may
	// currently have 0, 1, or more defers active).
	openDefer bool
	sp        uintptr // sp at time of defer
	pc        uintptr // pc at time of defer
	fn        func()  // can be nil for open-coded defers
    ...
}
```

倒着看这部分汇编会更容易理解:
- `callq	0x42f000 <runtime.deferprocStack>` 调用 deferprocStack
- `leaq	448(%rsp), %rax` 在调用前将参数保存到 rax
- `movq	%rcx, 472(%rsp)` 将 _defer 结构体 24~31th 字节设置为 rcx 的指, 即将 _defer.fn 设置为 rcx
- `leaq	136843(%rip), %rcx      # 0x49c4e8 <go:func.*+0x220>` 将 defer 函数的地址加载到 rcx

返回前调用 deferreturn 的汇编代码:
```
; 	return a + b
  47af8d: 48 8b 84 24 b0 02 00 00      	movq	688(%rsp), %rax     ; 将暂存在栈上的函数入参 a 和 b 存储到寄存器 rax 和 rcx
  47af95: 48 8b 8c 24 a8 02 00 00      	movq	680(%rsp), %rcx
  47af9d: 48 01 c8                     	addq	%rcx, %rax
  47afa0: 48 89 44 24 08               	movq	%rax, 8(%rsp)       ; 将结果暂存到栈上
  47afa5: e8 36 46 fb ff               	callq	0x42f5e0 <runtime.deferreturn>      ; 调用 deferreturn, 以 FILO 的顺序执行 defer
  47afaa: 48 8b 44 24 08               	movq	8(%rsp), %rax       ; 将暂存的返回值存储到 rax
  47afaf: 48 81 c4 98 02 00 00         	addq	$664, %rsp              # imm = 0x298   ; 释放申请的栈空间
  47afb6: 5d                           	popq	%rbp                                    ; 恢复 base pointer
  47afb7: c3                           	retq
```

## heap allocated


[Proposal: Low-cost defers through inline code, and extra funcdata to manage the panic case]: https://go.googlesource.com/proposal/+/refs/heads/master/design/34481-opencoded-defers.md
