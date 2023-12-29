在面试的过程中, 如果恰好遇到对方日常也使用 Go 做为主力语言,
我会选择一些简单而可扩展的问题交流下双方对 Go 的熟悉程度.

我喜欢的一个问题是让面试者告诉我下述代码的运行结果:
```go
func main() {
	for i := 0; i < 3; i++ {
		go func() {
			fmt.Println(i)
		}()
	}

	time.Sleep(time.Second)
}
```
正确的答案应该是: 乱序输出三个数字.
对于三种错误答案: 输出 1, 2, 3; 输出三个数字; 乱序输出 1, 2, 3; 都可以通过反问再给予一次机会.

进一步的, 我们可以询问如何让其至少将 1, 2, 3 都输出一次.
大多数时候, 我们的得到的答案会是将 i 做为参数传入.
此时我喜欢再追问, 下述代码中 `i := i` 的写法是否正确.
```go
func main() {
	for i := 0; i < 3; i++ {
		i := i
		go func() {
			fmt.Println(i)
		}()
	}

	time.Sleep(time.Second)
}
```
我并不认为这是一个 Language Lawyer 问题, 由于 Go 中 for 循环的特殊实现方式,
`i := i` 这种方式在 Go 中是普遍存在的.

极少数情况下, 我们可以再讨论下上述例子的原因, 允许面试者有更大的发挥机会.
其中包括的点有:
- Go 并不保证先启动的 goroutine 先执行
- Go 中 for 循环的实现是 one-instance-per-loop, 而不是 one-instance-per-iteration.

我们在下述例子中看到, i 和 v 的内存地址始终未曾改变:
```shell
~ cat main.go
func main() {
    nums := []int{1, 2, 3}
    for i, v := range nums {
        fmt.Println(&i, &v)
    }
}
~ go run main.go
0x1400009a018 0x1400009a020
0x1400009a018 0x1400009a020
0x1400009a018 0x1400009a020
```
- 闭包(closure)可能以值(by value)或者地址(by reference)的形式引用外部变量; 当引用 for 循环的中变量时, 是以地址的方式
- Go 允许在 inner block 中定义重名的变量, 下述代码虽然不好但合法
```shell
~ cat main.go | grep -A 7 "func fnVarScope"
func fnVarScope() {
    s := "hello world"
    {
        s := 10
        fmt.Println("s:", s)
    }
    fmt.Println("s:", s)
}
~ go run main.go
s: 10
s: hello world
```
