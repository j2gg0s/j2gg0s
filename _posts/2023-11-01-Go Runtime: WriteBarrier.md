在阅读 Go runtime 相关的代码时, 我们可以看到大量 write barrier 相关的代码或者注释,
大概可以猜到是和 GC 相关的, 具体用途和原理之前确并不知晓.

Go 的垃圾回收可以被简化为两个步骤: 标记(mark)和清除(sweep).
- 标记阶段, GC 将所有存活的对象标记为黑色.
- 清除阶段, GC 遍历内存将所有未被标记的对象回收.

GC 在标记阶段并不暂停所有协程的执行.
这就需要我们考虑, 如果对象的引用关系在标记阶段被修改了应该怎么办?

假设:
- t1时, 对象 A 被标记为黑色, 即 A 引用的对象已都被 GC 发现并处理.
- t2时, 用户新建了对象 C, 并将其赋值给 A 的字段 ref, 即 A.ref = C

由于 C 仅被 A 引用, 但 A 已被标记为黑色, 所以 GC 不会再去标记 C.
在随后的清除阶段, 虽然 C 依然被引用, 但是会因为未被标记而被 GC 的回收, 这显然时不可接受的.

为了处理这种情况, Go 引入 write barrier, 即由编译器在需要的地方插入相关代码处理.

我们构造一个相关的例子:
```go
var sink *int

func main() {
        foo := []int{1, 2, 3}
        sink = &foo[1]
}
```
在生成的汇编代码中我们可以找到相关内容:
```
cat -n objdump | grep -A 100 "main.main>:"
...
138038          sink = &foo[1]
138039    4576b1:       48 8d 48 08             lea    0x8(%rax),%rcx
138040    4576b5:       83 3d d4 10 0a 00 00    cmpl   $0x0,0xa10d4(%rip)        # 4f8790 <runtime.writeBarrier>
138041    4576bc:       74 15                   je     4576d3 <main.main+0x53>
138042    4576be:       66 90                   xchg   %ax,%ax
138043    4576c0:       e8 1b d2 ff ff          call   4548e0 <runtime.gcWriteBarrier2>
138044    4576c5:       49 89 0b                mov    %rcx,(%r11)
138045    4576c8:       48 8b 05 51 32 07 00    mov    0x73251(%rip),%rax        # 4ca920 <main.sink>
138046    4576cf:       49 89 43 08             mov    %rax,0x8(%r11)
138047    4576d3:       48 89 0d 46 32 07 00    mov    %rcx,0x73246(%rip)        # 4ca920 <main.sink>
...
```
其逻辑是:
- 通过全局变量 runtime.writeBarrier 判断是否开启了 write barrier
  [runtime.writeBarrier](https://github.com/golang/go/blob/go1.21.1/src/runtime/mgc.go#L215) 是一个全局变量,
  在进入标记段前开启, 进入清除阶段前关闭.

- 如果开启了, 则调用 runtime.gcWriteBarrier2 将对象保存到当前的 p, GMP 中的 p
  [runtime.gcWriteBarrier2](https://github.com/golang/go/blob/go1.21.1/src/runtime/asm_amd64.s#L1769)
  是直接以会编实现的函数, 会将寄存器 AX 内的指针保存到 p.wbbuf.

- 标记阶段结束时, GC 会额外处理这些对象
  在结束标记前, GC 会为调用 [wbBufFlush](https://github.com/golang/go/blob/go1.21.1/src/runtime/mwbbuf.go#L166) 处理这缓存的对象.
