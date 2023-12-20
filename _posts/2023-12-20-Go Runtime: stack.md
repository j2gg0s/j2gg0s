### Go's Entry Point
由于偷懒没有细看 Go 的编译逻辑, 所以在大多数时候我需要通过反汇编来确定一些事情, 比如说 Go 应用的入口.

Go 在 Linux 下的编译结果是 [ELF](https://en.wikipedia.org/wiki/Executable_and_Linkable_Format) 文件,
在 osx 中可以通过 brew 安装 readelf/objdump 等工具来解读.
```shell
~ brew search x86_64-linux-gnu-binutils
==> Formulae
x86_64-linux-gnu-binutils ✔                                                                     x86_64-elf-binutils
```

通过 ELF 的 File Header 确定程序的入口位置:
```shell
~ x86_64-linux-gnu-readelf -h main | grep Entry
  Entry point address:               0x455dc0
```

在 objdump 的输出中通过对应地址可以定位到入口函数是 `_rt0_amd64_linux`.
```shell
~ x86_64-linux-gnu-objdump -D -S main | grep -A 10 00455dc0
0000000000455dc0 <_rt0_amd64_linux>:
// license that can be found in the LICENSE file.

#include "textflag.h"

TEXT _rt0_amd64_linux(SB),NOSPLIT,$-8
        JMP     _rt0_amd64(SB)
  455dc0:       e9 bb e3 ff ff          jmp    454180 <_rt0_amd64>
  455dc5:       cc                      int3
  455dc6:       cc                      int3
  455dc7:       cc                      int3
```
结合[具体代码](https://github.com/golang/go/blob/go1.21.5/src/runtime/asm_amd64.s#L15)可以发现主要逻辑在 runtime.rt0_go.
其创建了最初的 m, 通过 [runtime.main](https://github.com/golang/go/blob/go1.21.5/src/runtime/proc.go#L144) 调用用户定义的 main 函数.

### stack
每个 goroutine 都拥有自己独立的栈, 初始空间小(2kb), 后续按需扩容.
g 除了保存当前栈的最高(stack.hi)和最低(stack.lo)地址外, 还使用 stackguard0 来记录应该扩容的地址.
考虑栈是从高位地址开始使用,
stackguard0 = stack.lo + stackGuard 等价于预留一部分空间, 避免扩容等情况时的栈溢出.

编译器会在可能需要扩容的函数调用中增加栈扩容的逻辑:
```shell
~ x86_64-linux-gnu-objdump -D -S main | cat -n - | grep -A 20 "main.main>:"
138032  0000000000457600 <main.main>:
138033
138034  func main() { add(3, 4) }
138035    457600:       49 3b 66 10             cmp    0x10(%r14),%rsp
138036    457604:       76 1d                   jbe    457623 <main.main+0x23>
138037    457606:       55                      push   %rbp
138038    457607:       48 89 e5                mov    %rsp,%rbp
138039    45760a:       48 83 ec 08             sub    $0x8,%rsp
138040    45760e:       b8 03 00 00 00          mov    $0x3,%eax
138041    457613:       bb 04 00 00 00          mov    $0x4,%ebx
138042    457618:       e8 c3 ff ff ff          call   4575e0 <main.add>
138043    45761d:       48 83 c4 08             add    $0x8,%rsp
138044    457621:       5d                      pop    %rbp
138045    457622:       c3                      ret
138046    457623:       e8 18 cf ff ff          call   454540 <runtime.morestack_noctxt.abi0>
138047    457628:       eb d6                   jmp    457600 <main.main>
138048
```
上述例子中:
- rsp 保存了当前函数栈的地址, 即 SP
- r14 保存了当前 g 的地址, 0x10 使其偏移 16 个字节, 对应 g.stackguard0
- L35 判断是否需要扩容
- L36 在需要扩容的时候跳转到 L46
- rutnime.morestack_noctxt 实现扩容
- L47 在扩容结束后跳转回函数入口

上述寄存器的含义可以参考 [Go internal ABI specification](https://go.googlesource.com/go/+/refs/heads/dev.regabi/src/cmd/compile/internal-abi.md#amd64-architecture).

栈的空间分配由 stackalloc 实现, 在其中我们可以看到一些熟悉的内容.
- 如果栈超过 32kb, 直接从 mheap 分配, 否则从 p 的缓存中分配.
- 同一个 mspan 内的栈大小相同, 以便以链表的形式维护空闲的栈空间.

当栈触发扩容或者缩容时, 运行时会分配一块新的空间并将内容都复制过去.
其中复杂的地方在于如何处理指向原始栈空间的指针.

由于编译器保证了[只有栈上的指针可以指向栈上的数据](https://docs.google.com/document/d/1wAaf1rYoM4S4gtnPh0zOlGzWtrZFQ5suE8qr2sD8uWQ/pub),
所以我们只需要遍历栈空间, 找到每一个指向栈的指针并做偏移调整即可.
特定地址是否时指针这一信息由 GC 维护, 我们直接在此处使用即可.

可能违反上述保证的情况, 比如 defer, panic 和 chan 等需要被特殊处理.
