### atomic
抛开 [Memory Consistency Model](https://research.swtch.com/mm) 不谈,
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
Store/Add/CompareAndSwap/Load 分别依赖 CPU 指令 XCHG/LOCK XADD/LOCK CMPXCHG/MOV 实现.

[LOCK](https://www.felixcloutier.com/x86/lock) 是加在特定指令前的前缀, 用于将对应指令转化为原子指令.
例如, [XADD](https://www.felixcloutier.com/x86/xadd) 将第一个参数和第二个参数相加后的值保存到第一个参数,
添加 LOCK 前缀后可以保证整个过程是排他且原子的.

[XCHG](https://www.felixcloutier.com/x86/xchg) 用于交换两个寄存器的值.
也可以用于交换寄存器和内存地址的值, 此时自带 LOCK 效果.

[CMPXCHG](https://www.felixcloutier.com/x86/cmpxchg) 借用寄存器 RAX 实现 CAS 效果.
如果 RAX 和第一个参数相等, 则将第二个参数的值赋给第一个参数.
否则将第一个参数赋给第二个参数.

[Intel® 64 and IA-32 Architectures Software Developer’s Manual](https://pdos.csail.mit.edu/6.828/2014/readings/ia32/IA32-3A.pdf) 中
Strengthening or Weakening the Memory Ordering Model 相关章节指明了 XCHG/LOCK 等相关指令会刷新缓存的写指令.
> Synchronization mechanisms in multiple-processor systems may depend upon a
> strong memory-ordering model. Here, a program can use a locking instruction such
> as the XCHG instruction or the LOCK prefix to insure that a read-modify-write operation on memory is carried out atomically.
> Locking operations typically operate like I/O operations in that they wait for all previous instructions to complete and for all
> buffered writes to drain to memory (see Section 7.1.2, “Bus Locking”).

因此 Store/Add/CompareAndSwap 等操作修改后直接可以被其他处理器看到.
同时, Go 也保证了内存数据的对齐.
那么, Load 操作就可以直接用 MOV 来实现.

## Memory Model
Memory Model 虽然是计算器中的一个重要概念, 但可能真的是绝大多数程序员无需关心的领域.
虽然我们可能都无法彻底理解, 但是花一定时间树立一个大体正确的概念是一件很值得的事情.
Russ Cox 为 Memory Model 写[三篇文章](https://research.swtch.com/mm)是比较好的入口.

首先, Memory Model 并不指如何管理内存, 其定义的是多处理系统中, 不同处理器之间如何共享和同步内存.
在下述例子中, 我们假设两个处理器分别运行如下的代码:
```
// Thread 1             // Thread 2
x = 1;                  r1 = x;
y = 1;                  r2 = y;
```
那么 (r1, r2) 可能的结果是 (0, 0), (1, 0), (1, 1), (0, 1).
对, 在没有明确 CPU 和编译器的 Memory Model 时, (0, 1) 也是有可能的.
CUP/编译器的重排指令都可能导致最终结果为 (0, 1).
而当我们约定 Memory Model 为 Sequential Consistency(SC) 时, r1=0&&r2=1 就是一种不可能的结果.
因为 SC 要求在所有处理器上观察到的内存操作顺序与单个处理器上的执行顺序一致.

在现实世界中, CPU 为了更高的性能, 会选择比 SC 更宽松的内存模型.
比如 x86 对应的就是 Total Store Order(TSO).
相对 SC, TSO 依然保证所有的写操作(Store)在所有处理器上观察到的顺序是一致的.
但 TSO 并不保证, 读(Load)和写(Store)在不同处理器上观察到的顺序是一致的.
在下面的例子中, r1=1&&r2=0 在 SC 中是不被允许的, 但在 TSO 中是可能出现的.
```
// Thread 1             // Thread 2
x = 1;                  y = 1;
r1 = y;                 r2 = x;
```
x86 等架构在实现 TSO 时, 基本都会为每个处理器维护一个 write buffer,
避免每个写操作都需要直接和内存交互, 进而提高 CPU 执行效率.
同时, x86 也提供了 FENCH 等内存屏障指令,
允许用户主动清空 write buffer, 确保指令前的写操作对所有处理器可见.

诸如 Java/C++/Go 等高级语言也会定义自己的的内存模型.
Go 是保证在没有数据竞争的情况下保证 SC, 即 data-race-free sequential-consistency(DRF-SC).

Data race 是指同时有两个以上的处理器访问同一个变量, 其中至少有一个是写操作.
在下述的代码中, 就包括 data race:
```go
package main

import "time"

var x int

func load() int {
	return x
}

func store(i int) {
	x = i
}

func main() {
	go store(32)
	go load()
	time.Sleep(1)
}
```
```shell
~ go build -race main.go
~ ./main
==================
WARNING: DATA RACE
Read at 0x000100a128a0 by goroutine 6:
  main.load()
      /Users/j2gg0s/go/src/github.com/j2gg0s/j2gg0s/examples/go-race/main.go:8 +0x28
  main.main.func2()
      /Users/j2gg0s/go/src/github.com/j2gg0s/j2gg0s/examples/go-race/main.go:17 +0x28

Previous write at 0x000100a128a0 by goroutine 5:
  main.store()
      /Users/j2gg0s/go/src/github.com/j2gg0s/j2gg0s/examples/go-race/main.go:12 +0x2c
  main.main.func1()
      /Users/j2gg0s/go/src/github.com/j2gg0s/j2gg0s/examples/go-race/main.go:16 +0x2c

Goroutine 6 (running) created at:
  main.main()
      /Users/j2gg0s/go/src/github.com/j2gg0s/j2gg0s/examples/go-race/main.go:17 +0x34

Goroutine 5 (finished) created at:
  main.main()
      /Users/j2gg0s/go/src/github.com/j2gg0s/j2gg0s/examples/go-race/main.go:16 +0x28
==================
Found 1 data race(s)
```
当存在 data race 时, Go 并不提供任何保证, 我们需要通过使用 lock/chan/atomic 等机制避免数据竞争.
一种方式是加锁保护数据.
```go
package main

import (
	"sync"
	"time"
)

var x int
var mu sync.Mutex

func load() int {
	mu.Lock()
	defer mu.Unlock()
	return x
}

func store(i int) {
	mu.Lock()
	defer mu.Unlock()
	x = i
}

func main() {
	go store(32)
	go load()
	time.Sleep(1)
}
```
另一种方式使用 atomic.
```
package main

import (
	"sync/atomic"
	"time"
)

var x atomic.Int64

func load() int64 {
	return x.Load()
}

func store(i int) {
	x.Store(int64(i))
}

func main() {
	go store(32)
	go load()
	time.Sleep(1)
}
```
