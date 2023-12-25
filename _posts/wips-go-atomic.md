抛开大 BOSS [Memory Consistency Model](https://research.swtch.com/mm) 不谈,
[sync/atomic](https://pkg.go.dev/sync/atomic) 的实现是简单而直观的.

我们首先构造一个案例, 基于 linux/amd64 编译后, 使用 objdump 查看汇编结果.
```go
import (
	"fmt"
	"sync/atomic"
)

var x atomic.Int64

func main() {
	x.Store(32)
	x.Add(1)
	x.CompareAndSwap(33, 34)
	i := x.Load()
	fmt.Println(i)
}
```
```shell
~ GOOS=linux GOARCH=amd64 go build main.go
~ x86_64-linux-gnu-objdump -D -S main | grep -A 65 "main.main>:" | cat -n -
 1	000000000047ad60 <main.main>:
 2		"sync/atomic"
 3	)
 4
 5	var x atomic.Int64
 6
 7	func main() {
 8	  47ad60:	49 3b 66 10          	cmp    0x10(%r14),%rsp
 9	  47ad64:	0f 86 81 00 00 00    	jbe    47adeb <main.main+0x8b>
10	  47ad6a:	55                   	push   %rbp
11	  47ad6b:	48 89 e5             	mov    %rsp,%rbp
12	  47ad6e:	48 83 ec 38          	sub    $0x38,%rsp
13		x.Store(32)
14	  47ad72:	90                   	nop
15
16	// Load atomically loads and returns the value stored in x.
17	func (x *Int64) Load() int64 { return LoadInt64(&x.v) }
18
19	// Store atomically stores val into x.
20	func (x *Int64) Store(val int64) { StoreInt64(&x.v, val) }
21	  47ad73:	b9 20 00 00 00       	mov    $0x20,%ecx
22	  47ad78:	48 8d 15 89 16 0d 00 	lea    0xd1689(%rip),%rdx        # 54c408 <main.x>
23	  47ad7f:	48 87 0a             	xchg   %rcx,(%rdx)
24		x.Add(1)
25	  47ad82:	90                   	nop
26	func (x *Int64) CompareAndSwap(old, new int64) (swapped bool) {
27		return CompareAndSwapInt64(&x.v, old, new)
28	}
29
30	// Add atomically adds delta to x and returns the new value.
31	func (x *Int64) Add(delta int64) (new int64) { return AddInt64(&x.v, delta) }
32	  47ad83:	b9 01 00 00 00       	mov    $0x1,%ecx
33	  47ad88:	f0 48 0f c1 0a       	lock xadd %rcx,(%rdx)
34		x.CompareAndSwap(33, 34)
35	  47ad8d:	90                   	nop
36		return CompareAndSwapInt64(&x.v, old, new)
37	  47ad8e:	b8 21 00 00 00       	mov    $0x21,%eax
38	  47ad93:	b9 22 00 00 00       	mov    $0x22,%ecx
39	  47ad98:	f0 48 0f b1 0a       	lock cmpxchg %rcx,(%rdx)
40	  47ad9d:	0f 94 c1             	sete   %cl
41		i := x.Load()
42	  47ada0:	90                   	nop
43	func (x *Int64) Load() int64 { return LoadInt64(&x.v) }
44	  47ada1:	48 8b 05 60 16 0d 00 	mov    0xd1660(%rip),%rax        # 54c408 <main.x>
45		fmt.Println(i)
46	  47ada8:	44 0f 11 7c 24 28    	movups %xmm15,0x28(%rsp)
47	  47adae:	e8 ed e9 f8 ff       	call   4097a0 <runtime.convT64>
48	  47adb3:	48 8d 0d 06 70 00 00 	lea    0x7006(%rip),%rcx        # 481dc0 <type:*+0x6dc0>
49	  47adba:	48 89 4c 24 28       	mov    %rcx,0x28(%rsp)
50	  47adbf:	48 89 44 24 30       	mov    %rax,0x30(%rsp)
51		return Fprintln(os.Stdout, a...)
52	  47adc4:	48 8b 1d 5d 37 0a 00 	mov    0xa375d(%rip),%rbx        # 51e528 <os.Stdout>
53	  47adcb:	48 8d 05 36 75 03 00 	lea    0x37536(%rip),%rax        # 4b2308 <go:itab.*os.File,io.Writer>
54	  47add2:	48 8d 4c 24 28       	lea    0x28(%rsp),%rcx
55	  47add7:	bf 01 00 00 00       	mov    $0x1,%edi
56	  47addc:	48 89 fe             	mov    %rdi,%rsi
57	  47addf:	90                   	nop
58	  47ade0:	e8 7b ae ff ff       	call   475c60 <fmt.Fprintln>
59	}
60	  47ade5:	48 83 c4 38          	add    $0x38,%rsp
61	  47ade9:	5d                   	pop    %rbp
62	  47adea:	c3                   	ret
63	func main() {
64	  47adeb:	e8 10 fc fd ff       	call   45aa00 <runtime.morestack_noctxt.abi0>
65	  47adf0:	e9 6b ff ff ff       	jmp    47ad60 <main.main>
66
```

runtime/internal/atomic 中的实现通过 inline 的形式直接嵌入到了 main.main 中, 稍显杂乱, 但是并不妨碍阅读.
其中:
- L23 的 Store 使用了 [XCHG](https://www.felixcloutier.com/x86/xchg)
xchg 是 AMD64 中用于交换两个寄存器或者寄存器和内存的指令, 当参数中含有内存地址时,
其会自动上锁. 我们可以直观的理解为原子的交换两个参数的值.

- L33 的 Add 使用了 LOCK XADD
xadd 结合 lock 能够保证读改写的操作是原子的.

- L39 的 CompareAndSwap 使用了 LOCK [CMPXCHG](https://www.felixcloutier.com/x86/cmpxchg)
cmpxchg 除了传入的两个参数, 还借用了寄存器 RAX.
如果 dst 和 rax 相同, 则将 src 赋给 dst, 否则将 dst 赋给 src.

- Load 最最直接, 只需要 MOV
Go 保证内存是对齐的, 那么 MOV 自身就是原子的.
