[Go 1.1 Function Calls]() 中介绍了函数调用在编译&汇编层面的是实现, 其中比较特别的是 `indirect call of func value`.
新手在不知道这个点的情况下去看相关的汇编时很容易被卡住.

我们以如下代码为例子:
```go
//go:noinline
func max(a, b int) int {
    if a > b {
        return a
    }
    return b
}

func main() {
    max(10, 20)

    imax := max
    imax(10, 20)

    x := 1
    y := 2
    iadd := func(a, b int) int {
        return x + y + a + b
    }
    iadd(10, 20)

    // 直接调用并不需要特殊实现, 即使是闭包
    func(a, b int) int {
        return x + y + a + b
    }(10, 20)
}
```

编译命令为 `GOOS=linux GOARCH=amd64 GOSSAFUNC=main.main go21 build -gcflags=-l main.go`, -l 用于告诉编译器不要进行 inline 优化.
反汇编命令为 `x86_64-linux-gnu-objdump -D -S main > objdump`.

main 函数对应的汇编:
```shell
cat -n objdump | grep main.main\>\: -A 65
138056	00000000004576a0 <main.main>:
138057	}
138058
138059	func main() {
138060	  4576a0:	49 3b 66 10          	cmp    0x10(%r14),%rsp
138061	  4576a4:	0f 86 97 00 00 00    	jbe    457741 <main.main+0xa1>
138062	  4576aa:	55                   	push   %rbp
138063	  4576ab:	48 89 e5             	mov    %rsp,%rbp
138064	  4576ae:	48 83 ec 38          	sub    $0x38,%rsp
138065		max(10, 20)
138066	  4576b2:	b8 0a 00 00 00       	mov    $0xa,%eax
138067	  4576b7:	bb 14 00 00 00       	mov    $0x14,%ebx
138068	  4576bc:	0f 1f 40 00          	nopl   0x0(%rax)
138069	  4576c0:	e8 bb ff ff ff       	call   457680 <main.max>
138070
138071		imax := max
138072		imax(10, 20)
138073	  4576c5:	48 8b 0d 74 49 01 00 	mov    0x14974(%rip),%rcx        # 46c040 <go:func.*+0x168>
138074	  4576cc:	b8 0a 00 00 00       	mov    $0xa,%eax
138075	  4576d1:	bb 14 00 00 00       	mov    $0x14,%ebx
138076	  4576d6:	48 8d 15 63 49 01 00 	lea    0x14963(%rip),%rdx        # 46c040 <go:func.*+0x168>
138077	  4576dd:	ff d1                	call   *%rcx
138078
138079		x := 1
138080		y := 2
138081		iadd := func(a, b int) int {
138082	  4576df:	44 0f 11 7c 24 20    	movups %xmm15,0x20(%rsp)
138083	  4576e5:	48 c7 44 24 30 00 00 	movq   $0x0,0x30(%rsp)
138084	  4576ec:	00 00
138085	  4576ee:	48 8d 0d 8b 00 00 00 	lea    0x8b(%rip),%rcx        # 457780 <main.main.func1>
138086	  4576f5:	48 89 4c 24 20       	mov    %rcx,0x20(%rsp)
138087	  4576fa:	48 c7 44 24 28 01 00 	movq   $0x1,0x28(%rsp)
138088	  457701:	00 00
138089	  457703:	48 c7 44 24 30 02 00 	movq   $0x2,0x30(%rsp)
138090	  45770a:	00 00
138091			return x + y + a + b
138092		}
138093		iadd(10, 20)
138094	  45770c:	48 8b 4c 24 20       	mov    0x20(%rsp),%rcx
138095	  457711:	b8 0a 00 00 00       	mov    $0xa,%eax
138096	  457716:	bb 14 00 00 00       	mov    $0x14,%ebx
138097	  45771b:	48 8d 54 24 20       	lea    0x20(%rsp),%rdx
138098	  457720:	ff d1                	call   *%rcx
138099
138100		func(a, b int) int {
138101			return x + y + a + b
138102		}(10, 20)
138103	  457722:	b8 01 00 00 00       	mov    $0x1,%eax
138104	  457727:	bb 02 00 00 00       	mov    $0x2,%ebx
138105	  45772c:	b9 0a 00 00 00       	mov    $0xa,%ecx
138106	  457731:	bf 14 00 00 00       	mov    $0x14,%edi
138107	  457736:	e8 25 00 00 00       	call   457760 <main.main.func2>
138108	}
138109	  45773b:	48 83 c4 38          	add    $0x38,%rsp
138110	  45773f:	5d                   	pop    %rbp
138111	  457740:	c3                   	ret
138112	func main() {
138113	  457741:	e8 9a ce ff ff       	call   4545e0 <runtime.morestack_noctxt.abi0>
138114	  457746:	e9 55 ff ff ff       	jmp    4576a0 <main.main>
```

### 直接调用
对 max 函数的直接调用是非常直观的, 对应 138066~138069 行.
首先将参数保存到两个寄存器, 再直接通过函数地址调用函数.

### 间接调用
但当我们将 max 赋值给一个变量再调用时, 即间接调用, 汇编代码就变得复杂起来了.

首先 rip 在 x64 中是一个非常特殊的寄存器, 永远等于下一个指令的地址.
所以 138073 行 `mov    0x14974(%rip),%rcx        # 46c040 <go:func.*+0x168>` 是将 0x46c040(0x4576cc+0x14964) 的内容保存到寄存器 rcx.

定位到 0x46c040, 可以发现其属于 .rodata, 保存的内容是 457680, 也就是 main.max 在汇编的中地址.
```shell
cat -n objdump | grep 46c040\:
170516    46c040:       80 76 45 00             xorb   $0x0,0x45(%rsi)

cat -n objdump | grep main.max\>\: -A 30
138019  0000000000457680 <main.max>:
138020  package main
138021
138022  //go:noinline
138023  func max(a, b int) int {
138024      if a > b {
138025    457680:       48 39 c3                cmp    %rax,%rbx
138026    457683:       7d 01                   jge    457686 <main.max+0x6>
138027          return a
138028    457685:       c3                      ret
138029      }
138030      return b
138031    457686:       48 89 d8                mov    %rbx,%rax
```

那么 138077 行 `call *%rcx` 即是直接通过地址来调用 max 函数.

从 [Go 1.1 Function Calls][] 中, 我们可以得知.
对于间接调用, 编译器会使用一块内存来保存函数地址和相关变量.
这么做主要是为了处理闭包, 即函数对外部变量的引用.
具体的前因后果可以参看原文.

这块内存的地址在调用函数前需要被保存到寄存器 rdx.

以 iadd(138079~138098) 为例, 上述逻辑会更为明显.

调用前需要在栈上分配 24 个字节, 0x20(%rsp) 用于保存函数地址,
0x28(%rsp) 和 0x30(%rsp) 用于保存引用的两个外部变量 x 和 y.
这块内存的地址随后又被保存到寄存器 rdx.

函数内部基于寄存器 rdx, 偏移 8 个字节读取 x, 偏移 16 个字节读取到 y.
```shell
cat -n objdump | grep main.main.func1\>\: -A 30
138164  0000000000457780 <main.main.func1>:
138165      iadd := func(a, b int) int {
138166    457780:       48 8b 4a 08             mov    0x8(%rdx),%rcx
138167          return x + y + a + b
138168    457784:       48 03 4a 10             add    0x10(%rdx),%rcx
138169    457788:       48 01 c1                add    %rax,%rcx
138170    45778b:       48 8d 04 0b             lea    (%rbx,%rcx,1),%rax
138171    45778f:       c3                      ret
```

[Go 1.1 Function Calls]: https://docs.google.com/document/d/1bMwCey-gmqZVTpRax-ESeVuZGmjwbocYs1iHplK-cjo/pub
