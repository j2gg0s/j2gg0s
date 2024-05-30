多态(polymorphism)或者泛型(generics)在此指的是让一个函数可以服务多种不同的类型.

## 实现
[The Generic Dilemma](https://research.swtch.com/generic) 指出了实现泛型(generic)的两种主流方式.
一是以C++ 为代表的 monomorphization 方案. 由编译器针对实际使用到每一类情况生成具体的代码.
代价是编译缓慢且执行文件臃肿.  另一种是以 Java 为代表的 boxing.
对象被分配在堆上, 函数之间仅传递指针. 运行根据具体的类型推导出具体的方法.
这种方案的代价是拖慢了运行效率.

Go 在 1.18 引入的方案, 官方名称是
[GCShape Stenciling with Dictionaries](https://go.googlesource.com/proposal/+/refs/heads/master/design/generics-implementation-gcshape.md),
简单理解就是 monoorphinzation 和 boxing 一起用.
是不是有点自己面向 OKR 做技术优化的感觉了.

Go 的这套方案可以简单的理解为:
- 对于不同 gcshape, 采用 C++ 方案, 由编译器创建多份代码, 即 stenciling
- 对于同一 gcshape, 采用 Java 方案, 在运行时根据路由表, 即 dicitionaries, 确定具体的方法
- gcshape 的定义: two concrete types are in the same gcshape grouping if and only if they have the same underlying type or they are both pointer types
    - 即要么有完全相同的底层实现, 如 `int`, `type MyInt int`
    - 或者是指针, 所有指针共享属于同一个 gcshape

```shell
➜  go-generic git:(main) ✗ cat -n main.go
     1  package main
     2
     3  //go:noinline
     4  func Sum[T interface{ ~int | ~int32 | ~int64 }](nums []T) (s T) {
     5          if len(nums) == 0 {
     6                  return
     7          }
     8          for _, n := range nums {
     9                  s += n
    10          }
    11          return
    12  }
    13
    14  func main() {
    15          Sum([]int{1, 2, 3})
    16  }
➜  go-generic git:(main) ✗ cat -n objdump  | grep "Sum(\[]int{1, 2, 3})" -A 10
112037          Sum([]int{1, 2, 3})
112038    0x4577ee              440f117c2420            MOVUPS X15, 0x20(SP)
112039    0x4577f4              440f117c2428            MOVUPS X15, 0x28(SP)
112040    0x4577fa              48c744242001000000      MOVQ $0x1, 0x20(SP)
112041    0x457803              48c744242802000000      MOVQ $0x2, 0x28(SP)
112042    0x45780c              48c744243003000000      MOVQ $0x3, 0x30(SP)
112043    0x457815              488d05b4690200          LEAQ main..dict.Sum[int](SB), AX
112044    0x45781c              488d5c2420              LEAQ 0x20(SP), BX
112045    0x457821              b903000000              MOVL $0x3, CX
112046    0x457826              4889cf                  MOVQ CX, DI
112047    0x457829              e812000000              CALL main.Sum[go.shape.int](SB)
```
在上述的例子中:
- L112047 显示编译时创建了针对特定 gcshape 的函数 main.Sum[go.shape.int]
- L112043 是在调用 Sum 前, 将路由表 main..dict.Sum[int] 作为第一个参数加载到寄存器 AX
- L112044~46 代表了实际参数, 一个 int 数组, 占用三个寄存器

泛型的主要处理逻辑在编译器, stenciling 和 dict 的主要内容都是编译器生产的.
由于对整个编译模块都不熟悉, 我们就不去代码里面扣逻辑了.
但是在 [Generics implementation - Dictionaries](https://go.googlesource.com/proposal/+/refs/heads/master/design/generics-implementation-dictionaries.md#generics-implementation-dictionaries) 的基础上,
结合汇编结果, 我们依然可以去理解&&验证这部分的逻辑.

我们首先针对相同的逻辑构造泛型和无泛型的实现, 然后通过比较, 了解泛型带来的一些改变.
```go
package main

//go:noinline
func Say[T interface{ Name() string }](v T) string {
    return v.Name() + " say hello"
}

type Foo struct{}

//go:noinline
func (Foo) Name() string { return "foo" }

type Bar struct{}

//go:noinline
func (Bar) Name() string { return "bar" }

//go:noinline
func FooSay(v Foo) string {
    return v.Name() + " say hello"
}

func main() {
    foo, bar := Foo{}, Bar{}

    Say(foo)
    FooSay(foo)

    Say(bar)
}
```

```shell
➜  go-generic git:(main) ✗ GOOS=linux GOARCH=amd64 go tool objdump -S main | cat -n - | grep -E "TEXT main.(Say|FooSay)" -A 16
112042  TEXT main.FooSay(SB) /Users/j2gg0s/go/src/github.com/j2gg0s/j2gg0s/examples/go-generic/main.go
112043  func FooSay(v Foo) string {
...
112049          return v.Name() + " say hello"
112050    0x45782e              e8adffffff              CALL main.Foo.Name(SB)
112051    0x457833              4889d9                  MOVQ BX, CX
112052    0x457836              488d3d8af40000          LEAQ 0xf48a(IP), DI
112053    0x45783d              be0a000000              MOVL $0xa, SI
112054    0x457842              4889c3                  MOVQ AX, BX
112055    0x457845              31c0                    XORL AX, AX
112056    0x457847              e854e1feff              CALL runtime.concatstring2(SB)
112057    0x45784c              4883c428                ADDQ $0x28, SP
112058    0x457850              5d                      POPQ BP
--
112087  TEXT main.Say[go.shape.struct {}](SB) /Users/j2gg0s/go/src/github.com/j2gg0s/j2gg0s/examples/go-generic/main.go
112088  func Say[T interface{ Name() string }](v T) string {
...
112094          return v.Name() + " say hello"
112095    0x4578ae              488b08                  MOVQ 0(AX), CX
112096    0x4578b1              4889c2                  MOVQ AX, DX
112097    0x4578b4              ffd1                    CALL CX
112098    0x4578b6              4889d9                  MOVQ BX, CX
112099    0x4578b9              488d3d07f40000          LEAQ 0xf407(IP), DI
112100    0x4578c0              be0a000000              MOVL $0xa, SI
112101    0x4578c5              4889c3                  MOVQ AX, BX
112102    0x4578c8              31c0                    XORL AX, AX
112103    0x4578ca              e8d1e0feff              CALL runtime.concatstring2(SB)
```
FooSay 的逻辑是很好理解的:
- L112050 调用 Foo.Name, 返回保存在寄存器 AX&BX 上, string 需要占用两个寄存器, 一个保存指针, 一个保存长度
- 接下来的几行都是为了调用 [func concatstring2(buf \*tmpBuf, a0, a1 string) string](https://github.com/golang/go/blob/go1.21.8/src/runtime/string.go#L59)
- L112051, 112054 将 Foo.Name 的返回转移到寄存器 BX 和 CX
- L112052, 112053 将 " say hello" 加载到寄存器 DI, SI
- L112055 清空寄存器 AX

对比之下, 我们可以发现 main.Say[go.shape.struct {}] 中的主要变化是调用 Foo.Name 的逻辑:
- 此时的调用参数变为两个 AX 保存了 dict, BX 保存了 foo
- L112095 中 0(AX) 是将 dict 的第一个字段转移到了 CX,
	- 从后面的 CALL CX, 我们可以推测这个字段存储 Foo.Name
- L112096 将 dict 暂存到了 DX, 为 L112097 的 CALL CX 做准备
- 后续的逻辑和 FooSay 一致

## 代价
[Generics can make your Go code slower](https://planetscale.com/blog/generics-can-make-your-go-code-slower)
深入而详细的讲述了 generic 带来的性能损失, 包括:
- 当使用指针作为参数时, 泛型相比 interface{} 多一次 deference
- 当使用不同于 type parameter 的 interface{} 作为参数时, 泛型函数会调用 runtime.assertI2I

这篇文章发表于 2022-03-30, 使用 go1.18,
在今日 2024-05-31, 使用 go1.21.8,
基本都无法复现, 大概率是已经被优化掉, 小概率是我菜, 没复现对.
但无论如何都不影响这是一篇写的很好的文章.

我们需要修改下之前的示例代码, 添上一些我们需要的场景.
- L35&L36 是用于区分指针调用, 在泛型函数中的逻辑.
- L39 是用于探索多包一层 interface{} 的影响.
```go
     1  package main
     2
     3  type Sayer interface {
     4          Name() string
     5  }
     6
     7  type SayerWithLastName interface {
     8          Name() string
     9          LastName() string
    10  }
    11
    12  //go:noinline
    13  func Say[T Sayer](v T) string {
    14          return v.Name() + " say hello"
    15  }
    16
    17  type User struct {
    18          name     string
    19          lastName string
    20  }
    21
    22  func (v *User) LastName() string { return v.lastName }
    23
    24  //go:noinline
    25  func (v *User) Name() string { return v.name }
    26
    27  //go:noinline
    28  func SayerSay(v Sayer) string {
    29          return v.Name() + " say hello"
    30  }
    31
    32  func main() {
    33          foo := &User{name: "foo", lastName: "unknown"}
    34
    35          SayerSay(foo)
    36          Say(foo)
    37
    38          i := SayerWithLastName(foo)
    39          Say(i)
    40
    41          Say(&Anonymous{})
    42  }
    43
    44  type Anonymous struct{}
    45
    46  func (v *Anonymous) Name() string { return "unknown" }
```

调用者的汇编如下:
- Say(i) 添加的额外参数比较多, 第一个是 dict, 第二个是 itab(用于 interface{} 的路由转发)
```shell
➜  go-generic git:(main) ✗ GOOS=linux GOARCH=amd64 go tool objdump -S main | cat -n - | grep "SayerSay(foo)" -A 12
112085          SayerSay(foo)
112086    0x4578a8              4889c3                  MOVQ AX, BX
112087    0x4578ab              488d05f66c0200          LEAQ go:itab.*main.User,main.Sayer(SB), AX
112088    0x4578b2              e849ffffff              CALL main.SayerSay(SB)
112089          Say(foo)
112090    0x4578b7              488d05926b0200          LEAQ main..dict.Say[*main.User](SB), AX
112091    0x4578be              488b5c2418              MOVQ 0x18(SP), BX
112092    0x4578c3              e898010000              CALL main.Say[go.shape.*uint8](SB)
112093          Say(i)
112094    0x4578c8              488d05916b0200          LEAQ main..dict.Say[main.SayerWithLastName](SB), AX
112095    0x4578cf              488d1d126d0200          LEAQ go:itab.*main.User,main.SayerWithLastName(SB), BX
112096    0x4578d6              488b4c2418              MOVQ 0x18(SP), CX
112097    0x4578db              0f1f440000              NOPL 0(AX)(AX*1)
```

我们可以看到 SayerSay(&foo) 和 Say(&foo) 基本没有区别, 都只有一次动态调用.
泛型函数中因为将 dict 作为第一个参数, 所以需要在 L112204 和 L112205 切换下寄存器.
```shell
➜  go-generic git:(main) ✗ GOOS=linux GOARCH=amd64 go tool objdump -S main | cat -n - | grep -E "TEXT main.SayerSay|TEXT main.Say\[go.shape.*uint8\]" -A 20
112037  TEXT main.SayerSay(SB) /Users/j2gg0s/go/src/github.com/j2gg0s/j2gg0s/examples/go-generic/main.go
112038  func SayerSay(v Sayer) string {
...
112046          return v.Name() + " say hello"
112047    0x457818              488b4818                MOVQ 0x18(AX), CX
112048    0x45781c              4889d8                  MOVQ BX, AX
112049    0x45781f              90                      NOPL
112050    0x457820              ffd1                    CALL CX
112051    0x457822              4889d9                  MOVQ BX, CX
112052    0x457825              488d3d78f60000          LEAQ 0xf678(IP), DI
112053    0x45782c              be0a000000              MOVL $0xa, SI
112054    0x457831              4889c3                  MOVQ AX, BX
112055    0x457834              31c0                    XORL AX, AX
112056    0x457836              e865e1feff              CALL runtime.concatstring2(SB)
112057    0x45783b              4883c428                ADDQ $0x28, SP
--
112195  TEXT main.Say[go.shape.*uint8](SB) /Users/j2gg0s/go/src/github.com/j2gg0s/j2gg0s/examples/go-generic/main.go
112196  func Say[T Sayer](v T) string {
...
112202          return v.Name() + " say hello"
112203    0x457a6e              488b08                  MOVQ 0(AX), CX
112204    0x457a71              4889c2                  MOVQ AX, DX
112205    0x457a74              4889d8                  MOVQ BX, AX
112206    0x457a77              ffd1                    CALL CX
112207    0x457a79              4889d9                  MOVQ BX, CX
112208    0x457a7c              488d3d21f40000          LEAQ 0xf421(IP), DI
112209    0x457a83              be0a000000              MOVL $0xa, SI
112210    0x457a88              4889c3                  MOVQ AX, BX
112211    0x457a8b              31c0                    XORL AX, AX
112212    0x457a8d              e80edffeff              CALL runtime.concatstring2(SB)
112213    0x457a92              4883c428                ADDQ $0x28, SP
112214    0x457a96              5d                      POPQ BP
112215    0x457a97              c3                      RET
```

Say(i) 是有点意思的:
- L112098 的调用显示 AX 是 dict, BX 是 itab, CX 才是 foo
- 那么 L112171 和 L112175 是针对泛型的一次方法路由, 调用的是 interface SayerWithLastname 的 Name 方法
- L112236 和 112238 是针对 interface{} 的一次方法路由
我们可以看到, 相比直接使用指针, 多一次方法路由, 但 runtime.assertI2I 已经不再需要了.
多的一次方法路由是因为 L112095 以 SayerWithLastName 作为 key 来从 dict 中获取信息.
```shell
➜  go-generic git:(main) ✗ GOOS=linux GOARCH=amd64 go tool objdump -S main | cat -n - | grep -E "Say\(i|TEXT main.Say\[go.shape.interface|TEXT main.SayerWithLastName" -A 20
112093          Say(i)
112094    0x4578c8              488d05916b0200          LEAQ main..dict.Say[main.SayerWithLastName](SB), AX
112095    0x4578cf              488d1d126d0200          LEAQ go:itab.*main.User,main.SayerWithLastName(SB), BX
112096    0x4578d6              488b4c2418              MOVQ 0x18(SP), CX
112097    0x4578db              0f1f440000              NOPL 0(AX)(AX*1)
112098    0x4578e0              e8fb000000              CALL main.Say[go.shape.interface { LastName() string; Name() string }](SB)
--
112161  TEXT main.Say[go.shape.interface { LastName() string; Name() string }](SB) /Users/j2gg0s/go/src/github.com/j2gg0s/j2gg0s/examples/go-generic/main.go
112162  func Say[T Sayer](v T) string {
...
112170          return v.Name() + " say hello"
112171    0x4579f8              488b30                  MOVQ 0(AX), SI
112172    0x4579fb              4889c2                  MOVQ AX, DX
112173    0x4579fe              4889d8                  MOVQ BX, AX
112174    0x457a01              4889cb                  MOVQ CX, BX
112175    0x457a04              ffd6                    CALL SI
112176    0x457a06              4889d9                  MOVQ BX, CX
112177    0x457a09              488d3d94f40000          LEAQ 0xf494(IP), DI
112178    0x457a10              be0a000000              MOVL $0xa, SI
112179    0x457a15              4889c3                  MOVQ AX, BX
112180    0x457a18              31c0                    XORL AX, AX
112181    0x457a1a              e881dffeff              CALL runtime.concatstring2(SB)
--
112224  TEXT main.SayerWithLastName.Name(SB) <autogenerated>
112225
...
112234    0x457ad7              4889442418              MOVQ AX, 0x18(SP)
112235    0x457adc              48895c2420              MOVQ BX, 0x20(SP)
112236    0x457ae1              488b4820                MOVQ 0x20(AX), CX
112237    0x457ae5              4889d8                  MOVQ BX, AX
112238    0x457ae8              ffd1                    CALL CX
112239    0x457aea              4883c408                ADDQ $0x8, SP
112240    0x457aee              5d                      POPQ BP
112241    0x457aef              c3                      RET
112242    0x457af0              4889442408              MOVQ AX, 0x8(SP)
112243    0x457af5              48895c2410              MOVQ BX, 0x10(SP)
112244    0x457afa              e841ccffff              CALL runtime.morestack_noctxt.abi0(SB)
```
