我们构造个非常简化的例子来看一些比较有意义的事情.
```go
package main

import (
    "testing"
)

func BenchmarkMapStringWithString(b *testing.B) {
    for i := 0; i < b.N; i++ {
        getByString(m, key)
    }
}

func BenchmarkMapStringWithBytes(b *testing.B) {
    for i := 0; i < b.N; i++ {
        getByBytes(m, key)
    }
}

var (
    m   = map[string]bool{"hello": true}
    key = []byte("hello")
)

//go:noinline
func getByString(m map[string]bool, key []byte) bool {
    k := string(key)
    return m[k]
}

//go:noinline
func getByBytes(m map[string]bool, key []byte) bool {
    return m[string(key)]
}
```

上述两个 benchmark 的逻辑其实是完全相同的, 但 getByBytes 会显著的快于 getByString.
```shell
✗ go test . --bench .
goos: darwin
goarch: arm64
pkg: github.com/j2gg0s/j2gg0s/examples/go-map-string-optimize
BenchmarkMapStringWithString-10         155190159                7.467 ns/op
BenchmarkMapStringWithBytes-10          231703806                5.156 ns/op
PASS
ok      github.com/j2gg0s/j2gg0s/examples/go-map-string-optimize        3.982s
```

这是因为 Go 的编译器有一些针对性的优化,
[cmd/gc: optimized map[string] lookup from []byte key](https://github.com/golang/go/issues/3512).
简单的说, 就是当你通过 bytes 去访问 map[string] 时, 编译器会省略将 bytes 转化为 string 的步骤.

我们首先看常规例子, getByString 的编译结果, 其:
- 首先调用 `slicebytetostring` 将 []byte 转换为 stirng
- 再调用 `mapaccess1_faststr` 访问 map[string]
```shell
go tool objdump main | grep -A 20 "TEXT main.getByString"
TEXT main.getByString(SB) /Users/j2gg0s/go/src/github.com/j2gg0s/j2gg0s/examples/go-map-string-optimize/main.go
  main.go:15            0x45d260                493b6610                CMPQ SP, 0x10(R14)
  main.go:15            0x45d264                763f                    JBE 0x45d2a5
  main.go:15            0x45d266                55                      PUSHQ BP
  main.go:15            0x45d267                4889e5                  MOVQ SP, BP
  main.go:15            0x45d26a                4883ec40                SUBQ $0x40, SP
  main.go:15            0x45d26e                48895c2458              MOVQ BX, 0x58(SP)
  main.go:17            0x45d273                4889442450              MOVQ AX, 0x50(SP)
  main.go:16            0x45d278                488d442420              LEAQ 0x20(SP), AX
  main.go:16            0x45d27d                0f1f00                  NOPL 0(AX)
  main.go:16            0x45d280                e87bc8feff              CALL runtime.slicebytetostring(SB)
  main.go:17            0x45d285                4889c1                  MOVQ AX, CX
  main.go:17            0x45d288                4889df                  MOVQ BX, DI
  main.go:17            0x45d28b                488d058e790000          LEAQ 0x798e(IP), AX
  main.go:17            0x45d292                488b5c2450              MOVQ 0x50(SP), BX
  main.go:17            0x45d297                e8a416fbff              CALL runtime.mapaccess1_faststr(SB)
  main.go:17            0x45d29c                0fb600                  MOVZX 0(AX), AX
  main.go:17            0x45d29f                4883c440                ADDQ $0x40, SP
  main.go:17            0x45d2a3                5d                      POPQ BP
  main.go:17            0x45d2a4                c3                      RET
  main.go:15            0x45d2a5                4889442408              MOVQ AX, 0x8(SP)
```
而触发了编译器优化的例子, getByBytes, 则不需要 slicebytetostring.
```shell
go tool objdump main | grep -A 20 "TEXT main.getByBytes"
TEXT main.getByBytes(SB) /Users/j2gg0s/go/src/github.com/j2gg0s/j2gg0s/examples/go-map-string-optimize/main.go
  main.go:21            0x45d2e0                493b6610                CMPQ SP, 0x10(R14)
  main.go:21            0x45d2e4                762b                    JBE 0x45d311
  main.go:21            0x45d2e6                55                      PUSHQ BP
  main.go:21            0x45d2e7                4889e5                  MOVQ SP, BP
  main.go:21            0x45d2ea                4883ec20                SUBQ $0x20, SP
  main.go:21            0x45d2ee                48895c2438              MOVQ BX, 0x38(SP)
  main.go:22            0x45d2f3                4889cf                  MOVQ CX, DI
  main.go:22            0x45d2f6                4889d9                  MOVQ BX, CX
  main.go:22            0x45d2f9                4889c3                  MOVQ AX, BX
  main.go:22            0x45d2fc                488d051d790000          LEAQ 0x791d(IP), AX
  main.go:22            0x45d303                e83816fbff              CALL runtime.mapaccess1_faststr(SB)
  main.go:22            0x45d308                0fb600                  MOVZX 0(AX), AX
  main.go:22            0x45d30b                4883c420                ADDQ $0x20, SP
  main.go:22            0x45d30f                5d                      POPQ BP
  main.go:22            0x45d310                c3                      RET
  main.go:21            0x45d311                4889442408              MOVQ AX, 0x8(SP)
  main.go:21            0x45d316                48895c2410              MOVQ BX, 0x10(SP)
  main.go:21            0x45d31b                48894c2418              MOVQ CX, 0x18(SP)
  main.go:21            0x45d320                48897c2420              MOVQ DI, 0x20(SP)
  main.go:21            0x45d325                e816ccffff              CALL runtime.morestack_noctxt.abi0(SB)
```

这种优化的前提是 Go 用个指向首地址的指针和长度来表示 string, 和 bytes 的表示方法基本相同.
[unsafe.String(ptr \*byte, len IntegerType) string](https://github.com/golang/go/blob/master/src/unsafe/unsafe.go#L264)
是有力的佐证.
