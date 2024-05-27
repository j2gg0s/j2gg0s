Rob Pike 在 [The Laws of Reflection](https://go.dev/blog/laws-of-reflection) 中总结了 Go 中有关 reflect 的三条法则:
1. Reflection goes from interface value to reflection object.
2. Reflection goes from reflection object to interface value.
3. To modify a reflection object, the value must be settable.

从其中第一条, 我们可以明确 Go 的反射是基于 interface 的.
[reflect]() 中的两个入口函数的参数也都是 interface:
[ValueOf(any) Value](https://github.com/golang/go/blob/go1.21.9/src/reflect/value.go#L3203),
[TypeOf(any) Type](https://github.com/golang/go/blob/go1.21.9/src/reflect/type.go#L1153).
我们在日常使用中对此感触不深是因为调用参数做了类型的隐式转换.

而 Go 能够在 interface 的基础上实现反射则是依赖 interface 即存储了值,
也存储了精确的类型信息. 参见 [reflect.Value](https://github.com/golang/go/blob/go1.21.9/src/reflect/value.go#L39).
```go
type Value struct {
    // typ_ holds the type of the value represented by a Value.
    // Access using the typ method to avoid escape of v.
    typ_ *abi.Type

    // Pointer-valued data or, if flagIndir is set, pointer to data.
    // Valid when either flagIndir is set or typ.pointers() is true.
    ptr unsafe.Pointer
```

我们先看 Go 是如何构造 interface.
观察下面对面对应的编译结果, 可以看到两次分别调用了 runtime.convT32 和 runtime.convT.
即 [runtime.convXXX](https://github.com/golang/go/blob/go1.21.9/src/runtime/iface.go#L313).
```go
package main

type Mather interface {
    Add(a, b int32) int32
    Sub(a, b int64) int64
}

type Inter struct{ id int32 }

//go:noinline
func (adder Inter) Add(a, b int32) int32 { return a + b }

//go:noinline
func (adder Inter) Sub(a, b int64) int64 { return a - b }

type Structer struct {
    x int32
    y string
}

//go:noinline
func (adder Structer) Add(a, b int32) int32 { return a + b }

//go:noinline
func (adder Structer) Sub(a, b int64) int64 { return a - b }

func main() {
    // NOTE: avoid devirtualizing
    for _, m := range []Mather{Mather(Inter{id: 6754}), Mather(Structer{7, "8"})} {
        m.Add(10, 32)
    }
}
```
```shell
~ GOOS=linux GOARCH=amd64 go tool compile -S main.go | cat -n - | grep "runtime.convT32(SB)" -B 3 -A 10
    63        0x0049 00073 (/Users/j2gg0s/go/src/github.com/j2gg0s/j2gg0s/examples/go-interface-reflect/main.go:29)    MOVL    $6754, main..autotmp_6+20(SP)
    64        0x0051 00081 (/Users/j2gg0s/go/src/github.com/j2gg0s/j2gg0s/examples/go-interface-reflect/main.go:29)    MOVL    main..autotmp_6+20(SP), AX
    65        0x0055 00085 (/Users/j2gg0s/go/src/github.com/j2gg0s/j2gg0s/examples/go-interface-reflect/main.go:29)    PCDATA    $1, $1
    66        0x0055 00085 (/Users/j2gg0s/go/src/github.com/j2gg0s/j2gg0s/examples/go-interface-reflect/main.go:29)    CALL    runtime.convT32(SB)
    67        0x005a 00090 (/Users/j2gg0s/go/src/github.com/j2gg0s/j2gg0s/examples/go-interface-reflect/main.go:29)    LEAQ    go:itab.<unlinkable>.Inter,<unlinkable>.Mather(SB), CX
    68        0x0061 00097 (/Users/j2gg0s/go/src/github.com/j2gg0s/j2gg0s/examples/go-interface-reflect/main.go:29)    MOVQ    CX, main..autotmp_5+64(SP)
    69        0x0066 00102 (/Users/j2gg0s/go/src/github.com/j2gg0s/j2gg0s/examples/go-interface-reflect/main.go:29)    MOVQ    AX, main..autotmp_5+72(SP)
    70        0x006b 00107 (/Users/j2gg0s/go/src/github.com/j2gg0s/j2gg0s/examples/go-interface-reflect/main.go:29)    LEAQ    type:<unlinkable>.Structer(SB), AX
    71        0x0072 00114 (/Users/j2gg0s/go/src/github.com/j2gg0s/j2gg0s/examples/go-interface-reflect/main.go:29)    LEAQ    main..autotmp_2+40(SP), BX
    72        0x0077 00119 (/Users/j2gg0s/go/src/github.com/j2gg0s/j2gg0s/examples/go-interface-reflect/main.go:29)    PCDATA    $1, $2
    73        0x0077 00119 (/Users/j2gg0s/go/src/github.com/j2gg0s/j2gg0s/examples/go-interface-reflect/main.go:29)    CALL    runtime.convT(SB)
    74        0x007c 00124 (/Users/j2gg0s/go/src/github.com/j2gg0s/j2gg0s/examples/go-interface-reflect/main.go:29)    LEAQ    go:itab.<unlinkable>.Structer,<unlinkable>.Mather(SB), CX
    75        0x0083 00131 (/Users/j2gg0s/go/src/github.com/j2gg0s/j2gg0s/examples/go-interface-reflect/main.go:29)    MOVQ    CX, main..autotmp_5+80(SP)
    76        0x0088 00136 (/Users/j2gg0s/go/src/github.com/j2gg0s/j2gg0s/examples/go-interface-reflect/main.go:29)    MOVQ    AX, main..autotmp_5+88(SP)
```
在编译结果中, 可以看到:
- runtime.convT32 接受 uint32 作为参数, 类型 `go:itab.<unlinkable>.Inter,<unlinkable>.Mather` 和返回的值存储在栈上 64(SP)
- runtime.convT 接受 `type:<unlinkable>.Structer` 作为第一个参数, 类型 `go:itab.<unlinkable>.Structer,<unlinkable>.Matcher(SB)` 和返回的值存储在栈上 81(SP).

在 reflect.TypeOf 中, 将 interface 的 typ 和 value 组装成 emptyinterface.
```go
// TypeOf returns the reflection Type that represents the dynamic type of i.
// If i is a nil interface value, TypeOf returns nil.
func TypeOf(i any) Type {
    eface := *(*emptyInterface)(unsafe.Pointer(&i))
    // Noescape so this doesn't make i to escape. See the comment
    // at Value.typ for why this is safe.
    return toType((*abi.Type)(noescape(unsafe.Pointer(eface.typ))))
}
...
// emptyInterface is the header for an interface{} value.
type emptyInterface struct {
    typ  *abi.Type
    word unsafe.Pointer
}
```
