系统调用(System Call)是内核提供给用户的功能.
在 Linux 上, 需要通过在汇编层面调用指令 SYSCALL,
并按约定将参数传入特定的寄存器.
可以阅读 [LINUX SYSTEM CALL TABLE FOR X86 64](https://blog.rchapman.org/posts/Linux_System_Call_Table_for_x86_64/)
有一个更直观的认识.

触发系统调用时, 需要从用户态(user mode)切换到内核态(kernal mode).
系统调用期间, 发起调用的线程会被挂起, 操作系统会将 CPU 分配给其他线程.
调用完成后, 操作系统会在合适的时机重新执行挂起的线程.

Go 并没有直接使用操作系统的进程&线程模型, 而是提出了自己的 GMP 模型, 并实现了运行时的调度器.
为了提高执行效率和延迟, 在系统调用前, Go 需要将 m 和 p 解绑, 允许其他空闲的 m 去执行 p.
在系统调用后, Go 需要将 m 和 p 重新绑定, 恢复执信后续指令.

为了处理上述逻辑, Go 在系统调用前后增加了相关逻辑, [Syscall](https://github.com/golang/go/blob/go1.21.1/src/syscall/syscall_linux.go#L68).
```go
func Syscall(trap, a1, a2, a3 uintptr) (r1, r2 uintptr, err Errno) {
	runtime_entersyscall()
	// N.B. Calling RawSyscall here is unsafe with atomic coverage
	// instrumentation and race mode.
	//
	// Coverage instrumentation will add a sync/atomic call to RawSyscall.
	// Race mode will add race instrumentation to sync/atomic. Race
	// instrumentation requires a P, which we no longer have.
	//
	// RawSyscall6 is fine because it is implemented in assembly and thus
	// has no coverage instrumentation.
	//
	// This is typically not a problem in the runtime because cmd/go avoids
	// adding coverage instrumentation to the runtime in race mode.
	r1, r2, err = RawSyscall6(trap, a1, a2, a3, 0, 0, 0)
	runtime_exitsyscall()
	return
}
```
RawSyscall6 是对汇编的简单封装
[asm_linux_amd64.s](https://github.com/golang/go/blob/go1.21.1/src/runtime/internal/syscall/asm_linux_amd64.s):
```
// func Syscall6(num, a1, a2, a3, a4, a5, a6 uintptr) (r1, r2, errno uintptr)
//
// We need to convert to the syscall ABI.
//
// arg | ABIInternal | Syscall
// ---------------------------
// num | AX          | AX
// a1  | BX          | DI
// a2  | CX          | SI
// a3  | DI          | DX
// a4  | SI          | R10
// a5  | R8          | R8
// a6  | R9          | R9
//
// r1  | AX          | AX
// r2  | BX          | DX
// err | CX          | part of AX
//
// Note that this differs from "standard" ABI convention, which would pass 4th
// arg in CX, not R10.
TEXT ·Syscall6<ABIInternal>(SB),NOSPLIT,$0
	// a6 already in R9.
	// a5 already in R8.
	MOVQ	SI, R10 // a4
	MOVQ	DI, DX  // a3
	MOVQ	CX, SI  // a2
	MOVQ	BX, DI  // a1
	// num already in AX.
	SYSCALL
	CMPQ	AX, $0xfffffffffffff001
	JLS	ok
	NEGQ	AX
	MOVQ	AX, CX  // errno
	MOVQ	$-1, AX // r1
	MOVQ	$0, BX  // r2
	RET
ok:
	// r1 already in AX.
	MOVQ	DX, BX // r2
	MOVQ	$0, CX // errno
	RET
```

`runtime.entersyscall` 和 `runtime.exitsyscall` 是我们这次关注的重点.
我们需要首先简单介绍下 GMP, 详细的介绍可以自行 Google 或者参考
[Scheduling In Go : Part II - Go Scheduler](https://www.ardanlabs.com/blog/2018/08/scheduling-in-go-part2.html).

GMP 中 m 是操作系统线程, p 代表资源, g 是 goroutine, 代表被执行的代码.
每个 p 使用队列 runq 保存待执行的 goroutine.

`entersyscall` 相对简单, 核心逻辑是需要将 m 和 p 解绑,
这是因为执行系统调用时, 系统线程 m 会被挂起, 如果 m 和 p 不解绑, 则 p 关联的 g 在此期间都无法被执行.
具体而言:
- 将 g 的状态从 running 变更为 syscall
- m 和 p 解绑, m.oldp 设置为 p
- p 的状态更改为 syscall

`exitsyscall` 相对复杂, 优先尝试恢复执行之前代码, 否则尝试将 m 和其他闲置的 p 关联,
都失败的话则 m 变为闲置状态, g 被放入全局队列 globalrunq.
具体而言:
- 如果 oldp 的状态依然为 syscall 或者存在闲置的 p, 则
    - 将 oldp/p 与 m 绑定, 并将 p 的状态设置为 running.
    - 将 g 的状态从 syscall 变为 running, 并执行后续代码
    - 上述两种情况下, g 的状态从 syscall 变为 running, m 直接执行 g
- 切换到 g0 执行下述逻辑
    - g 的状态从 syscall 变为 running, 并解除和 m 的绑定
    - 如果有闲置的 p, 则将 m 和 p 绑定, 并且执行执行 g
    - 否则将 g 放到 globalrunq 后挂起 m, 等待被唤醒
