最近看到有人在讨论下面的函数返回.
```go
func fnReturn() (ret int) {
    v := 10
    defer func() {
        ret += 1
    }()
    return v
}
```

其实我也不是太确定,
一是我不会这么写, 避免自己给自己增加工作量,
二是最近这半年没花时间在 Go 上, 不太确定 Go 会怎么处理.

但是因为闲, 所以我还是想去看看.
和往常一样, 我们直接从汇编结果来看, 避免去翻代码.

我们准备的测试代码如下:
```go
package main

import "fmt"

func main() {
    fmt.Println(fn())
    fmt.Println(fnReturn())
}

func fn() int {
    v := 10
    defer func() {
        v += 1
    }()
    return v
}

func fnReturn() (ret int) {
    v := 10
    defer func() {
        ret += 1
    }()
    return v
}
```
运行结果是
```shell
✗ go run main.go
10
11
```
获取对应的编译结果:
```shell
✗ GOOS=linux GOARCH=amd64 go build main.go
✗ go tool objdump main > objdump
```

先看 `fnReturn`,
```shell
cat -n objdump | grep -A 50 "TEXT main.fnReturn"
122696  TEXT main.fnReturn(SB) /Users/j2gg0s/go/src/github.com/j2gg0s/j2gg0s/examples/go-defer-return/main.go
122697    main.go:18            0x480a60                493b6610                CMPQ SP, 0x10(R14)
122698    main.go:18            0x480a64                767b                    JBE 0x480ae1
122699    main.go:18            0x480a66                55                      PUSHQ BP
122700    main.go:18            0x480a67                4889e5                  MOVQ SP, BP
122701    main.go:18            0x480a6a                4883ec28                SUBQ $0x28, SP
122702    main.go:18            0x480a6e                66440fd67c2420          MOVQ X15, 0x20(SP)
122703    main.go:18            0x480a75                c644240700              MOVB $0x0, 0x7(SP)
122704    main.go:18            0x480a7a                48c744240800000000      MOVQ $0x0, 0x8(SP)  // ret 初始化
122705    main.go:20            0x480a83                440f117c2410            MOVUPS X15, 0x10(SP)
122706    main.go:20            0x480a89                488d0570000000          LEAQ main.fnReturn.func1(SB), AX
122707    main.go:20            0x480a90                4889442410              MOVQ AX, 0x10(SP)
122708    main.go:20            0x480a95                488d442408              LEAQ 0x8(SP), AX    // ret 的地址加载到寄存器 AX
122709    main.go:20            0x480a9a                4889442418              MOVQ AX, 0x18(SP)   // ret 的地址保存到栈
122710    main.go:20            0x480a9f                488d442410              LEAQ 0x10(SP), AX
122711    main.go:20            0x480aa4                4889442420              MOVQ AX, 0x20(SP)
122712    main.go:20            0x480aa9                c644240701              MOVB $0x1, 0x7(SP)
122713    main.go:23            0x480aae                48c74424080a000000      MOVQ $0xa, 0x8(SP)  // ret 的值从 0 变为 10
122714    main.go:23            0x480ab7                c644240700              MOVB $0x0, 0x7(SP)
122715    main.go:23            0x480abc                488b542420              MOVQ 0x20(SP), DX
122716    main.go:23            0x480ac1                488b02                  MOVQ 0(DX), AX
122717    main.go:23            0x480ac4                ffd0                    CALL AX             // 调用 defer
122718    main.go:23            0x480ac6                488b442408              MOVQ 0x8(SP), AX    // 返回 ret
122719    main.go:23            0x480acb                4883c428                ADDQ $0x28, SP
122720    main.go:23            0x480acf                5d                      POPQ BP
122721    main.go:23            0x480ad0                c3                      RET
122722    main.go:23            0x480ad1                e8ca15fbff              CALL runtime.deferreturn(SB)
122723    main.go:23            0x480ad6                488b442408              MOVQ 0x8(SP), AX
122724    main.go:23            0x480adb                4883c428                ADDQ $0x28, SP
122725    main.go:23            0x480adf                5d                      POPQ BP
122726    main.go:23            0x480ae0                c3                      RET
122727    main.go:18            0x480ae1                e8fafafdff              CALL runtime.morestack_noctxt.abi0(SB)
122728    main.go:18            0x480ae6                e975ffffff              JMP main.fnReturn(SB)
122729
122730  TEXT main.fnReturn.func1(SB) /Users/j2gg0s/go/src/github.com/j2gg0s/j2gg0s/examples/go-defer-return/main.go
122731    main.go:20            0x480b00                488b4208                MOVQ 0x8(DX), AX
122732    main.go:21            0x480b04                48ff00                  INCQ 0(AX)
122733    main.go:22            0x480b07                c3                      RET
```

在看 `fn`, 其级别逻辑是相同的, 区别的点在于:
`fn` 中有两个变量, 0x8(sp) 和 0x10(sp), 前者用于保存会烦的值, 后者对应 v, defer 操作的是 v 而不是 0x8(sp).
而 `fnReturn` 中二者被编译优化成了个一个变量 0x8(sp).
```shell
✗ cat -n objdump | grep -A 50 "TEXT main.fn("
122654  TEXT main.fn(SB) /Users/j2gg0s/go/src/github.com/j2gg0s/j2gg0s/examples/go-defer-return/main.go
122655    main.go:10            0x4809a0                493b6610                CMPQ SP, 0x10(R14)
122656    main.go:10            0x4809a4                0f8686000000            JBE 0x480a30
122657    main.go:10            0x4809aa                55                      PUSHQ BP
122658    main.go:10            0x4809ab                4889e5                  MOVQ SP, BP
122659    main.go:10            0x4809ae                4883ec30                SUBQ $0x30, SP
122660    main.go:10            0x4809b2                66440fd67c2428          MOVQ X15, 0x28(SP)
122661    main.go:10            0x4809b9                c644240700              MOVB $0x0, 0x7(SP)
122662    main.go:10            0x4809be                48c744240800000000      MOVQ $0x0, 0x8(SP)
122663    main.go:11            0x4809c7                48c74424100a000000      MOVQ $0xa, 0x10(SP)
122664    main.go:12            0x4809d0                440f117c2418            MOVUPS X15, 0x18(SP)
122665    main.go:12            0x4809d6                488d0563000000          LEAQ main.fn.func1(SB), AX
122666    main.go:12            0x4809dd                4889442418              MOVQ AX, 0x18(SP)
122667    main.go:12            0x4809e2                488d442410              LEAQ 0x10(SP), AX
122668    main.go:12            0x4809e7                4889442420              MOVQ AX, 0x20(SP)
122669    main.go:12            0x4809ec                488d442418              LEAQ 0x18(SP), AX
122670    main.go:12            0x4809f1                4889442428              MOVQ AX, 0x28(SP)
122671    main.go:12            0x4809f6                c644240701              MOVB $0x1, 0x7(SP)
122672    main.go:15            0x4809fb                488b442410              MOVQ 0x10(SP), AX
122673    main.go:15            0x480a00                4889442408              MOVQ AX, 0x8(SP)
122674    main.go:15            0x480a05                c644240700              MOVB $0x0, 0x7(SP)
122675    main.go:15            0x480a0a                488b542428              MOVQ 0x28(SP), DX
122676    main.go:15            0x480a0f                488b02                  MOVQ 0(DX), AX
122677    main.go:15            0x480a12                ffd0                    CALL AX
122678    main.go:15            0x480a14                488b442408              MOVQ 0x8(SP), AX
122679    main.go:15            0x480a19                4883c430                ADDQ $0x30, SP
122680    main.go:15            0x480a1d                5d                      POPQ BP
122681    main.go:15            0x480a1e                c3                      RET
122682    main.go:15            0x480a1f                90                      NOPL
122683    main.go:15            0x480a20                e87b16fbff              CALL runtime.deferreturn(SB)
122684    main.go:15            0x480a25                488b442408              MOVQ 0x8(SP), AX
122685    main.go:15            0x480a2a                4883c430                ADDQ $0x30, SP
122686    main.go:15            0x480a2e                5d                      POPQ BP
122687    main.go:15            0x480a2f                c3                      RET
122688    main.go:10            0x480a30                e8abfbfdff              CALL runtime.morestack_noctxt.abi0(SB)
122689    main.go:10            0x480a35                e966ffffff              JMP main.fn(SB)
122690
122691  TEXT main.fn.func1(SB) /Users/j2gg0s/go/src/github.com/j2gg0s/j2gg0s/examples/go-defer-return/main.go
122692    main.go:12            0x480a40                488b4208                MOVQ 0x8(DX), AX
122693    main.go:13            0x480a44                48ff00                  INCQ 0(AX)
122694    main.go:14            0x480a47                c3                      RET
```


基于此, 我们可以构造一个更具体的例子:
```shell
✗ cat main.go
package main

import "fmt"

func main() {
        fmt.Println(fn())
        fmt.Println(fnReturn())
}

func fn() int {
        v := 10
        defer func() {
                v += 1
                fmt.Println("v", v)
        }()
        return v
}

func fnReturn() (ret int) {
        v := 10
        defer func() {
                ret += 1
                fmt.Println("v", v)
        }()
        return v
}
✗ go run main.go
v 11
10
v 10
11
```
如果你再去看 fnReturn 的编译结果, 它现在应用两个变量分别保存 v 和返回值了.
