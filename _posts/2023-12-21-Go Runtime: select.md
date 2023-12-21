Goroutine, chan, select 是在 Go 中使用并发的关键.

### 编译器的工作
[select](https://go.dev/ref/spec#Select_statements) 经过编译后会调用
[runtime.selectgo](https://github.com/golang/go/blob/go1.21.5/src/runtime/select.go#L121),
其函数签名如下:
```go
// selectgo implements the select statement.
//
// cas0 points to an array of type [ncases]scase, and order0 points to
// an array of type [2*ncases]uint16 where ncases must be <= 65536.
// Both reside on the goroutine's stack (regardless of any escaping in
// selectgo).
//
// For race detector builds, pc0 points to an array of type
// [ncases]uintptr (also on the stack); for other builds, it's set to
// nil.
//
// selectgo returns the index of the chosen scase, which matches the
// ordinal position of its respective select{recv,send,default} call.
// Also, if the chosen scase was a receive operation, it reports whether
// a value was received.
func selectgo(cas0 *scase, order0 *uint16, pc0 *uintptr, nsends, nrecvs int, block bool) (int, bool) {
```
只有单个 case 的 select 会变转变成 if, 只有单个 case 和 default 的 select 会变转变成 if-else.
其他情况, 编译器首先会将 case 翻译成 scase, 其占用 16 字节, 包含一个指向 chan 的指针和一个指向收发元素的指针.
order0 是调用前分配的, 用于 selectgo 保存对 scase 的遍历顺序, nsends 和 nrecvs 分别对应 sendch 和 recvch 的数量,
block 代表是否存在 default.

我们以如下代码来验证理解:
```go
//go:noinline
func fnSelect(r1, r2, s chan int) (int, bool) {
	var i, z int
	var ok bool
	select {
	case i, ok = <-r1:
		z = 1 + i
	case i, ok = <-r2:
		z = 2 + i
	case s <- 0:
		z, ok = 0, true
	default:
	}
	return z, ok
}

func main() {
	r1, r2, s := make(chan int), make(chan int), make(chan int)
	fnSelect(r1, r2, s)
	close(r1)
	close(r2)
	close(s)
}
```
依然是先以 linux/amd64 为目标平台编译后, 用 objdump 反汇编后查看 fnSelect.
对应的编译命令为 `GOOS=linux GOARCH=amd64 go build main.go`,
汇编结果为:
```shell
~ x86_64-linux-gnu-objdump -D -S main | grep "main.fnSelect>:" -A 90 | cat -n
     1  0000000000458d60 <main.fnSelect>:
     2  package main
     3
     4  //go:noinline
     5  func fnSelect(r1, r2, s chan int) (int, bool) {
     6    458d60:       4c 8d 64 24 f0          lea    -0x10(%rsp),%r12
     7    458d65:       4d 3b 66 10             cmp    0x10(%r14),%r12
     8    458d69:       0f 86 b6 00 00 00       jbe    458e25 <main.fnSelect+0xc5>
     9    458d6f:       55                      push   %rbp
    10    458d70:       48 89 e5                mov    %rsp,%rbp
    11    458d73:       48 81 ec 88 00 00 00    sub    $0x88,%rsp
    12          var i, z int
    13          var ok bool
    14          select {
    15    458d7a:       48 c7 44 24 30 00 00    movq   $0x0,0x30(%rsp)
    16    458d81:       00 00
    17    458d83:       44 0f 11 7c 24 58       movups %xmm15,0x58(%rsp)
    18    458d89:       44 0f 11 7c 24 68       movups %xmm15,0x68(%rsp)
    19    458d8f:       44 0f 11 7c 24 78       movups %xmm15,0x78(%rsp)
    20          case i, ok = <-r1:
    21    458d95:       48 89 44 24 78          mov    %rax,0x78(%rsp)
    22    458d9a:       48 8d 54 24 40          lea    0x40(%rsp),%rdx
    23    458d9f:       48 89 94 24 80 00 00    mov    %rdx,0x80(%rsp)
    24    458da6:       00
    25                  z = 1 + i
    26          case i, ok = <-r2:
    27    458da7:       48 89 5c 24 68          mov    %rbx,0x68(%rsp)
    28    458dac:       48 8d 54 24 38          lea    0x38(%rsp),%rdx
    29    458db1:       48 89 54 24 70          mov    %rdx,0x70(%rsp)
    30                  z = 2 + i
    31          case s <- 0:
    32    458db6:       48 89 4c 24 58          mov    %rcx,0x58(%rsp)
    33    458dbb:       48 8d 54 24 30          lea    0x30(%rsp),%rdx
    34    458dc0:       48 89 54 24 60          mov    %rdx,0x60(%rsp)
    35          select {
    36    458dc5:       48 8d 44 24 58          lea    0x58(%rsp),%rax
    37    458dca:       48 8d 5c 24 4c          lea    0x4c(%rsp),%rbx
    38    458dcf:       31 c9                   xor    %ecx,%ecx
    39    458dd1:       bf 01 00 00 00          mov    $0x1,%edi
    40    458dd6:       be 02 00 00 00          mov    $0x2,%esi
    41    458ddb:       45 31 c0                xor    %r8d,%r8d
    42    458dde:       66 90                   xchg   %ax,%ax
    43    458de0:       e8 9b 54 fe ff          call   43e280 <runtime.selectgo>
    44                  z, ok = 0, true
    45          default:
    46    458de5:       48 85 c0                test   %rax,%rax
    47    458de8:       7d 06                   jge    458df0 <main.fnSelect+0x90>
    48    458dea:       31 c9                   xor    %ecx,%ecx
    49    458dec:       31 db                   xor    %ebx,%ebx
    50    458dee:       eb 29                   jmp    458e19 <main.fnSelect+0xb9>
    51          case s <- 0:
    52    458df0:       75 0e                   jne    458e00 <main.fnSelect+0xa0>
    53    458df2:       31 c9                   xor    %ecx,%ecx
    54    458df4:       bb 01 00 00 00          mov    $0x1,%ebx
    55    458df9:       eb 1e                   jmp    458e19 <main.fnSelect+0xb9>
    56    458dfb:       0f 1f 44 00 00          nopl   0x0(%rax,%rax,1)
    57          case i, ok = <-r2:
    58    458e00:       48 83 f8 01             cmp    $0x1,%rax
    59    458e04:       75 0b                   jne    458e11 <main.fnSelect+0xb1>
    60    458e06:       48 8b 4c 24 38          mov    0x38(%rsp),%rcx
    61                  z = 2 + i
    62    458e0b:       48 83 c1 02             add    $0x2,%rcx
    63          case i, ok = <-r2:
    64    458e0f:       eb 08                   jmp    458e19 <main.fnSelect+0xb9>
    65          case i, ok = <-r1:
    66    458e11:       48 8b 4c 24 40          mov    0x40(%rsp),%rcx
    67                  z = 1 + i
    68    458e16:       48 ff c1                inc    %rcx
    69          }
    70          return z, ok
    71    458e19:       48 89 c8                mov    %rcx,%rax
    72    458e1c:       48 81 c4 88 00 00 00    add    $0x88,%rsp
    73    458e23:       5d                      pop    %rbp
    74    458e24:       c3                      ret
    75  func fnSelect(r1, r2, s chan int) (int, bool) {
    76    458e25:       48 89 44 24 08          mov    %rax,0x8(%rsp)
    77    458e2a:       48 89 5c 24 10          mov    %rbx,0x10(%rsp)
    78    458e2f:       48 89 4c 24 18          mov    %rcx,0x18(%rsp)
    79    458e34:       e8 67 ce ff ff          call   455ca0 <runtime.morestack_noctxt.abi0>
    80    458e39:       48 8b 44 24 08          mov    0x8(%rsp),%rax
    81    458e3e:       48 8b 5c 24 10          mov    0x10(%rsp),%rbx
    82    458e43:       48 8b 4c 24 18          mov    0x18(%rsp),%rcx
    83    458e48:       e9 13 ff ff ff          jmp    458d60 <main.fnSelect>
```
快速理解上述代码的一种方式是从 L43 对 runtime.selectgo 的调用开始.
基于 [Go internal ABI specification](https://go.googlesource.com/go/+/refs/heads/dev.regabi/src/cmd/compile/internal-abi.md#amd64-architecture), 结合 selectgo 的参数签名,
我们知道参数被保存到寄存器 rax(cas0), rbx(order0), ecx, edi(nsends), esi(nrecvs) 和 r8d(block).

在上述例子中, cas0 应该是保存了三个 scase 的数组, L36 将栈上 0x58 加载到了寄存器 rax, 所以 cas0 的第一个元素保存在栈上 0x58. 又因为 scase 占用 16 字节, 所以 cas0 的第二和第三个元素保存在 0x68 和 0x78.

此时再去理解 L21~L34 就会简单的多, L21 将 fnSelect 的入参 chan r1 保存到 0x68, L22&L23 指明用 0x38 来保存对应的元素.

L39 nsends 是 0x1, L40 nrecvs 是 2, L41 block 是 false.

### selectgo 的逻辑
select 处理多个 chan 和 send/recv 处理单个 chan 的逻辑大体是一致的.
当 send 时:
1. 如果有等待接受元素的 g, 则直接将元素交给 g
2. 否则, 如果 chan buf 中还有空间, 则将元素放到 buf
3. 否则, 将自己加入 chan 的发送等待队列 sendq, 并让出执行权
当 recv 时:
1. 如果有等待发送元素的 g, 则直接从其获取元素
2. 否则, 如果 chan buf 中有缓存的元素, 则从其中获取元素
3. 否则, 将自己加入 chan 的接受等待队列 recvq, 并让出执行权

select 额外的工作包括:
- 随机 case 的执行顺序
- lock/unlock 从单个 ch 变成多个
- g 需要加入多个 ch 的 sendq/recvq
- g 的等待列表 waiting 也需要加所有 ch 加入
