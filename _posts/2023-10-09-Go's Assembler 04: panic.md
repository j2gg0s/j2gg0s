Go 并没有 try-catch 这样的语法, 捕获 panic, 依赖 defer 和 recovery 的配合.

我们首先明确, panic 是通过 runtime.gopanic 实现的.
runtime.gopanic 唯一的参数类型是 any, 需要占用两个寄存器来传递参数.
```
192471          panic(nil)
192472    47ae0e:       31 c0                   xor    %eax,%eax
192473    47ae10:       31 db                   xor    %ebx,%ebx
192474    47ae12:       e8 e9 52 fb ff          call   430100 <runtime.gopanic>
```

当前 defer 有三种实现机制, open-coded, stack-allocated 和 heap-allocated.
stack-allocated 和 heap-allocated 在处理 panic 时逻辑基本等同.

stack-allocated 首先通过调用 runtime.deferprocStack 将 defer 函数保存到 goroutine,
随后在函数返回前插入对 runtime.deferreturn 的调用, 以先进后出的方式执行插入的 defer 函数.
在处理 panic 时, 仅需要直接遍历 goroutine 中保存相关记录的变量即可.

open-coded 通过在汇编中直接插入调用来大幅提高 defer 的性能,
代价之一是复杂化了 panic 的处理. 具体可以参见
[Proposal: Low-cost defers through inline code, and extra funcdata to manage the panic case](https://go.googlesource.com/proposal/+/refs/heads/master/design/34481-opencoded-defers.md).

open-coded 方案下, 编译器需要用一块内存保存了:
- deferBits, 每个 defer 函数是否已经被执行
- nDefers, defer 函数的数量
- 每个 defer 函数和参数的地址
这块内存的地址, 做为 FUNCDATA 保存.

在遇到 panic 时, Go 会遍历 stack 上的每个函数, 如果对应 FUNCDATA 存在,
则将 open defer 加入 goroutine 的 defer 链表,
等待后续一起执行.

我并没有找到具体处理 FUNCDATA 的逻辑, 但我们依然能够从其他方面来验证理解.

首先是在处理 panic 时, 遍历获取 open defer 的代码, [addOneOpenDeferFrame](https://github.com/golang/go/blob/go1.21.1/src/runtime/panic.go#L657):
```go
for u.initAt(pc, uintptr(sp), 0, gp, 0); u.valid(); u.next() {
    frame := &u.frame
    ...
    f := frame.fn
    fd := funcdata(f, abi.FUNCDATA_OpenCodedDeferInfo)
    if fd == nil {
        continue
    }
    ...
}
```
其次是执行 open defer 的相关代码, [runOpenDeferFrame](https://github.com/golang/go/blob/go1.21.1/src/runtime/panic.go#L749)
```go
func runOpenDeferFrame(d *_defer) bool {
	done := true
	fd := d.fd

	deferBitsOffset, fd := readvarintUnsafe(fd)
	nDefers, fd := readvarintUnsafe(fd)
	deferBits := *(*uint8)(unsafe.Pointer(d.varp - uintptr(deferBitsOffset)))

	for i := int(nDefers) - 1; i >= 0; i-- {
		// read the funcdata info for this defer
		var closureOffset uint32
		closureOffset, fd = readvarintUnsafe(fd)
		if deferBits&(1<<i) == 0 {
			continue
		}
		closure := *(*func())(unsafe.Pointer(d.varp - uintptr(closureOffset)))
		d.fn = closure
		deferBits = deferBits &^ (1 << i)
		*(*uint8)(unsafe.Pointer(d.varp - uintptr(deferBitsOffset))) = deferBits
		p := d._panic
		// Call the defer. Note that this can change d.varp if
		// the stack moves.
		deferCallSave(p, d.fn)
		if p != nil && p.aborted {
			break
		}
		d.fn = nil
		if d._panic != nil && d._panic.recovered {
			done = deferBits == 0
			break
		}
	}

	return done
}
```
最后是处理 open defer 的汇编代码中将函数地址和参数塞入到栈中的连续区块:
```shell
192493          a := 10
192494          b := 20
192495          c := 30
192496
192497          defer func() {
192498    47ae43:       44 0f 11 7c 24 28       movups %xmm15,0x28(%rsp)
192499    47ae49:       48 8d 05 f0 01 00 00    lea    0x1f0(%rip),%rax        # 47b040 <main.main.func1>
192500    47ae50:       48 89 44 24 28          mov    %rax,0x28(%rsp)
192501    47ae55:       48 c7 44 24 30 0a 00    movq   $0xa,0x30(%rsp)
192502    47ae5c:       00 00
192503    47ae5e:       48 8d 44 24 28          lea    0x28(%rsp),%rax
192504    47ae63:       48 89 44 24 48          mov    %rax,0x48(%rsp)
192505    47ae68:       c6 44 24 07 01          movb   $0x1,0x7(%rsp)
192506                  fmt.Println(a)
192507          }()
```

recover 对应的是 runtime.gorecover, 函数会返回保存在 goroutine 的 _panic 变量.
```
192589                  r := recover()
192590    47af40:       e8 fb 58 fb ff          call   430840 <runtime.gorecover>
```
需要注意的是, recover 之后, 需要调用 deferreturn, 执行剩余的 defer 函数.
这一步通过两点实现.
首先编译时, 在函数返回后塞入对 deferreturn 的调用.
```
192528                  if r != nil {
192529                          fmt.Println(r)
192530                  }
192531                  fmt.Println(c)
192532          }()
192533          doPanic()
192534    47aec1:       e8 3a ff ff ff          call   47ae00 <main.doPanic>
192535  }
192536    47aec6:       c6 44 24 07 03          movb   $0x3,0x7(%rsp)
192537    47aecb:       48 8b 54 24 38          mov    0x38(%rsp),%rdx
192538    47aed0:       48 8b 02                mov    (%rdx),%rax
192539    47aed3:       ff d0                   call   *%rax
192540    47aed5:       c6 44 24 07 01          movb   $0x1,0x7(%rsp)
192541    47aeda:       48 8b 54 24 40          mov    0x40(%rsp),%rdx
192542    47aedf:       48 8b 02                mov    (%rdx),%rax
192543    47aee2:       ff d0                   call   *%rax
192544    47aee4:       c6 44 24 07 00          movb   $0x0,0x7(%rsp)
192545    47aee9:       48 8b 54 24 48          mov    0x48(%rsp),%rdx
192546    47aeee:       48 8b 02                mov    (%rdx),%rax
192547    47aef1:       ff d0                   call   *%rax
192548    47aef3:       48 83 c4 50             add    $0x50,%rsp
192549    47aef7:       5d                      pop    %rbp
192550    47aef8:       c3                      ret
192551    47aef9:       e8 e2 46 fb ff          call   42f5e0 <runtime.deferreturn>
192552    47aefe:       48 83 c4 50             add    $0x50,%rsp
192553    47af02:       5d                      pop    %rbp
192554    47af03:       c3                      ret
```
随后运行时, 将 defer 的变量 pc 设置为上述 deferreturn 的地址, 在 recover 后跳转到这个指令, [addOneOpenDeferFrame](https://github.com/golang/go/blob/go1.21.1/src/runtime/panic.go#L702)
```go
// These are the pc/sp to set after we've
// run a defer in this frame that did a
// recover. We return to a special
// deferreturn that runs any remaining
// defers and then returns from the
// function.
d1.pc = frame.fn.entry() + uintptr(frame.fn.deferreturn)
```
