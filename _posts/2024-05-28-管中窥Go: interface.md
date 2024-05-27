如未做特别说明, 全文依然假设 GOOS=linux GOARCH=amd64, go 的版本为 go1.21.8.

## 值调用指针方法的实现
这大概率是一个语法糖, 由编译器塞入取地址指令.
我们构造了如下的例子, 编译后通过汇编来验证我们的猜想.
```go
     1    package main
     2    
     3    type Foo struct {
     4        name string
     5        age  int
     6    }
     7    
     8    //go:noinline
     9    func (foo *Foo) ChangeName(name string) {
    10        foo.name = name
    11    }
    12    
    13    func main() {
    14        foo := Foo{name: "foo", age: 35}
    15        foo.ChangeName("bar")
    16        (&foo).ChangeName("baz")
    17    }
```
```shell
➜  go-generic git:(main) ✗ GOOS=linux GOARCH=amd64 go build value_call_ptr_method.go
➜  go-generic git:(main) ✗ GOOS=linux GOARCH=amd64 go tool objdump -S value_call_ptr_method | grep "foo.ChangeName(\"bar" -A 11
        foo.ChangeName("bar")
  0x45786c              488d442418              LEAQ 0x18(SP), AX
  0x457871              488d1d9dea0000          LEAQ 0xea9d(IP), BX
  0x457878              b903000000              MOVL $0x3, CX
  0x45787d              0f1f00                  NOPL 0(AX)
  0x457880              e85bffffff              CALL main.(*Foo).ChangeName(SB)
        (&foo).ChangeName("baz")
  0x457885              488d442418              LEAQ 0x18(SP), AX
  0x45788a              488d1d87ea0000          LEAQ 0xea87(IP), BX
  0x457891              b903000000              MOVL $0x3, CX
  0x457896              e845ffffff              CALL main.(*Foo).ChangeName(SB)
}
```
我们可以看到两次调用的汇编几乎是一致, 其中:
- 0x45786c 将 foo 的地址加载到寄存器 AX, 调用方法时, 需要将接受者作为一个参数传入.
- 0x457871 和 0x457878 将字符串 bar 加载到寄存器 BX, CX, 字符串需要用连个寄存器
- 0x45787d 是 NOPL 指令, 没有实际影响
- 0x457880 调用 ChangeName, 两个参数依次被保存在 AX, BX+CX

## interface 的实现
Russ Cox 的 [Go Data Structures: Interfaces](https://research.swtch.com/interfaces) 是了解 interface 实现的最好入口之一.
在此基础上, 我们通过一些构造的例子来加深/验证自己的理解.

```go
     1    package main
     2    
     3    import (
     4        "unsafe"
     5    )
     6    
     7    type Namer interface {
     8        GetName() string
     9    }
    10    
    11    type User struct {
    12        Name string
    13        Age  int
    14    }
    15    
    16    func (u User) GetName() string { return u.Name }
    17    
    18    //go:noinline
    19    func getName(i Namer) string { return i.GetName() }
    20    
    21    func main() {
    22        u := User{"Foo", 35}
    23        i := Namer(u)
    24        getName(i)
    25    }
```
上述代码对应的汇编如下:
```shell
➜  go-generic git:(main) ✗ GOOS=linux GOARCH=amd64 go tool compile -S itf.go | sed 's/\/Users\/j2gg0s\/go\/src\/github.com\/j2gg0s\/j2gg0s\/examples\/go-generic\///g' | cat -n - | grep -E "itf.go:(22|23|24)"
    65          0x000e 00014 (itf.go:22)        MOVQ    $0, main.u+16(SP)
    66          0x0017 00023 (itf.go:22)        MOVUPS  X15, main.u+24(SP)
    67          0x001d 00029 (itf.go:22)        LEAQ    go:string."Foo"(SB), CX
    68          0x0024 00036 (itf.go:22)        MOVQ    CX, main.u+16(SP)
    69          0x0029 00041 (itf.go:22)        MOVQ    $3, main.u+24(SP)
    70          0x0032 00050 (itf.go:22)        MOVQ    $35, main.u+32(SP)
    71          0x003b 00059 (itf.go:23)        LEAQ    type:<unlinkable>.User(SB), AX
    72          0x0042 00066 (itf.go:23)        LEAQ    main.u+16(SP), BX
    73          0x0047 00071 (itf.go:23)        PCDATA  $1, $0
    74          0x0047 00071 (itf.go:23)        CALL    runtime.convT(SB)
    75          0x004c 00076 (itf.go:24)        MOVQ    AX, BX
    76          0x004f 00079 (itf.go:24)        LEAQ    go:itab.<unlinkable>.User,<unlinkable>.Namer(SB), AX
    77          0x0056 00086 (itf.go:24)        CALL    main.getName(SB)
```
其中:
- L67~L70 新建了变量 u 并存放在 16(SP)
- L71 将类型 User 的加载到寄存器 AX
- L72 将变量 u 的地址加载到寄存器 BX
- L74 调用 runtime.convT, 两个入参保存在 AX 和 BX
- L75 将 runtime.convT 的返回从寄存器 AX 移动到寄存器 BX
- L76 将 interface 的 itab 加载到寄存器 AX
- L77 调用 main.getName

结合上述的汇编代码和 runtime, 不难理解 interface 在 runtime 中对应的结构体
[iface](https://github.com/golang/go/blob/go1.21.8/src/runtime/runtime2.go#L204).
```go
type iface struct {
    tab  *itab
    data unsafe.Pointer
}
...
type itab struct {
    inter *interfacetype
    _type *_type
    hash  uint32 // copy of _type.hash. Used for type switches.
    _     [4]byte
    fun   [1]uintptr // variable sized. fun[0]==0 means _type does not implement inter.
}
```
直观的来看, iface 保存的核心信息是:
- interface 的类型, itab.inter
- 底层的精确类型, itab.\_type
- 底层的值, data

L76 的 `go:itab.User,.Namer` 大概率是编译器结合 interface 和 struct 构造的 itab,
但是 `go tool compile` 并没有直接给出可以验证这一点的内容.
我们参考 [go-internal](https://github.com/teh-cmc/go-internals/blob/master/chapter2_interfaces/README.md#reconstructing-an-itab-from-an-executable)
, 尝试从 elf 文件中读取读取相关内容.
```shell
➜  go-generic git:(main) ✗ GOOS=linux GOARCH=amd64 go build itf.go
➜  go-generic git:(main) ✗
➜  go-generic git:(main) ✗ x86_64-linux-gnu-objdump -t -j .rodata itf | grep Namer
000000000047e3a8 g     O .rodata        0000000000000020 go:itab.main.User,main.Namer
➜  go-generic git:(main) ✗ x86_64-linux-gnu-objdump -t -j .rodata itf | grep go:itab.main.User,main.Namer | awk '{print "ibase=16;"toupper($1)}' | bc
4711336
➜  go-generic git:(main) ✗ x86_64-linux-gnu-objdump -t -j .rodata itf | grep go:itab.main.User,main.Namer | awk '{print "ibase=16;"toupper($5)}' | bc
32
➜  go-generic git:(main) ✗
➜  go-generic git:(main) ✗ x86_64-linux-gnu-readelf -St -W itf | grep -A 1 .rodata | tail -n +2
       PROGBITS        0000000000458000 058000 0272c6 00   0   0 32
➜  go-generic git:(main) ✗ x86_64-linux-gnu-readelf -St -W itf | grep -A 1 .rodata | tail -n +2 | awk '{print "ibase=16;"toupper($3)}' | bc
360448
➜  go-generic git:(main) ✗ x86_64-linux-gnu-readelf -St -W itf | grep -A 1 .rodata | tail -n +2 | awk '{print "ibase=16;"toupper($2)}' | bc
4554752
```
1. 我们首先将代码编译到指定平台
2. 随后读取到 go:itab.main.User,main.Namer 的地址和长度: 4711336(0x47e3a8) 和 32(0x20).
3. 为了将地址转换到 elf 文件内的偏移量, 读取 .rodata 的偏移量和地址: 360448(0x58000) 和 4554752(0x458000)

所以 go:itab.main.User,main.Namer 应该在文件的第 4711336-4554752+360448=517032 开始的 32 个字节.
```shell
➜  go-generic git:(main) ✗ dd if=itf of=/dev/stdout bs=1 count=32 skip=517032 2>/dev/null | hexdump
0000000 e920 0045 0000 0000 1160 0046 0000 0000
0000010 2a0b 99d4 0000 0000 7940 0045 0000 0000
0000020
```
通过 itab.hash 可以验证上述数据的准确性:
```go
    22    func main() {
    23        u := User{"Foo", 35}
    24        i := Namer(u)
    25        getName(i)
    26    
    27        iface := (*iface)(unsafe.Pointer(&i))
    28        fmt.Printf("%#x\n", iface.tab.hash)
    29    }
    30    
    31    // simplified definitions of runtime's iface & itab types
    32    type iface struct {
    33        tab *struct {
    34            inter uintptr
    35            _type uintptr
    36            hash  uint32
    37            _     [4]byte
    38            fun   [1]uintptr
    39        }
    40        data unsafe.Pointer
    41    }
```
运行结果 0x99d42a0b 和第 24 到 32 字节的内容完全相符.

## interface 的代价
比较直观的一点是, 在通过
[runtime.convT](https://github.com/golang/go/blob/go1.21.8/src/runtime/iface.go#L322) 将值转换为 iface.data 时,
可能需要分配一个堆上对象.
```shell
➜  go-generic git:(main) ✗ cat -n itf_test.go
     1  package main
     2
     3  import (
     4          "fmt"
     5          "io"
     6          "testing"
     7  )
     8
     9  func BenchmarkInterfaceAllocate(b *testing.B) {
    10          u := User{"Foo", 35}
    11
    12          var v interface{}
    13          for i := 0; i < b.N; i++ {
    14                  v = interface{}(u)
    15          }
    16
    17          fmt.Fprintln(io.Discard, v)
    18  }
    19
    20  func BenchmarkInterfaceNoAllocate(b *testing.B) {
    21          var v interface{}
    22          for i := 0; i < b.N; i++ {
    23                  v = interface{}(32)
    24          }
    25
    26          fmt.Fprintln(io.Discard, v)
    27  }
    28
    29  func BenchmarkInterfacePtr(b *testing.B) {
    30          u := &User{"Foo", 35}
    31
    32          var v interface{}
    33          for i := 0; i < b.N; i++ {
    34                  v = interface{}(u)
    35          }
    36
    37          fmt.Fprintln(io.Discard, v)
    38  }
➜  go-generic git:(main) ✗ go test -benchmem --bench=. ./...
goos: darwin
goarch: arm64
pkg: github.com/j2gg0s/j2gg0s/examples/go-generic
BenchmarkInterfaceAllocate-10           62783398                19.01 ns/op           24 B/op          1 allocs/op
BenchmarkInterfaceNoAllocate-10         1000000000               0.2959 ns/op          0 B/op          0 allocs/op
BenchmarkInterfacePtr-10                1000000000               0.2911 ns/op          0 B/op          0 allocs/op
PASS
ok      github.com/j2gg0s/j2gg0s/examples/go-generic    2.834s
```

复杂的是 interface 的 method dispatch.
从 Russ Cox 的文章中, 我们可以理解转发表保存在 itab.fun, 并由 runtime 在运行时构建.
但是从之前的例子来看, 依然存在的困惑点:
- 有谁触发并构建了 itab.fun
- 调用 main.getName 时并不能看到相关逻辑, 是编译器针对此类 case 直接填充了?

针对前一个问题, 我们构建一个 interface2interface 的例子来触发相关逻辑.
```shell
➜  go-generic git:(main) ✗ cat -n i2i.go
     1  package main
     2
     3  type Namer interface {
     4          GetName() string
     5  }
     6
     7  type NamerAndAger interface {
     8          GetName() string
     9          GetAge() int
    10  }
    11
    12  type User struct {
    13          Name string
    14          Age  int
    15  }
    16
    17  func (u User) GetName() string { return u.Name }
    18  func (u User) GetAge() int     { return u.Age }
    19
    20  //go:noinline
    21  func getName(i Namer) string { return i.GetName() }
    22
    23  func main() {
    24          u := NamerAndAger(User{"Foo", 35})
    25          getName(u)
    26  }
➜  go-generic git:(main) ✗ GOOS=linux GOARCH=amd64 go tool compile -S i2i.go | sed 's/\/Users\/j2gg0s\/go\/src\/github.com\/j2gg0s\/j2gg0s\/examples\/go-generic\///g' | cat -n - | grep "i2i.go:25"
    87          0x0051 00081 (i2i.go:25)        LEAQ    go:itab.<unlinkable>.User,<unlinkable>.NamerAndAger(SB), BX
    88          0x0058 00088 (i2i.go:25)        LEAQ    type:<unlinkable>.Namer(SB), AX
    89          0x005f 00095 (i2i.go:25)        PCDATA  $1, $1
    90          0x005f 00095 (i2i.go:25)        NOP
    91          0x0060 00096 (i2i.go:25)        CALL    runtime.convI2I(SB)
    92          0x0065 00101 (i2i.go:25)        MOVQ    main..autotmp_7+16(SP), BX
    93          0x006a 00106 (i2i.go:25)        PCDATA  $1, $0
    94          0x006a 00106 (i2i.go:25)        CALL    main.getName(SB)
```
此时, 我们不再调用 convT, 而是调用 [runtime.convI2I](https://github.com/golang/go/blob/go1.21.8/src/runtime/iface.go#L412).
其内部会调用 [itab.init](https://github.com/golang/go/blob/go1.21.8/src/runtime/iface.go#L76) 构建 itab.fun.

对于后一个问题, 我们依然可以通过读取 elf 中的内容来验证.
回顾 go.itab.main.User,main.Namer 其值, 除去用于对其的 4 个字节,
fun 的值是 7940 0045, 对应偏移为 0x457940.
毫不意外, 其是 User.GetName 的入口.
```shell
➜  go-generic git:(main) ✗ dd if=itf of=/dev/stdout bs=1 count=32 skip=517032 2>/dev/null | hexdump
0000000 e920 0045 0000 0000 1160 0046 0000 0000
0000010 2a0b 99d4 0000 0000 7940 0045 0000 0000
0000020
➜  go-generic git:(main) ✗ x86_64-linux-gnu-objdump -t -j .text itf | grep 457940
0000000000457940 g     F .text  0000000000000037 main.(*User).GetName
```

