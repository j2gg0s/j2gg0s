## 字符集和编码方式
字符集(character set)是一组字符(character)的集合.
简单的如 ASCII 字符集定义了英语中常见的 128 个字符.
复杂的如 Unicode 字符集定义了超过 14 万个字符, 囊括了绝大多数语言文字的需求.
Unicode 为每一个字符分配一个唯一的 code point, 从 0 到 0x10FFFF, 通常表示为 U+X,
X 为 4 到 6 个十六进制数字, 如英语字符 A 对应 U+0041.

编码(encoding)是指如何将字符集中的字符转化为计算机可以理解的二进制形式.
UTF-8 和 UTF-16 是最常见的两种编码方式, 针对的都是 Unicode 字符集.

UTF-8 用一到四个字节来表示一个 Unicode 字符 code point.
在 UTF-8 中, 127 个 ASCII 字符仅需一个字节来存储,
使得其在英语环境中非常高效, 同时页完全兼容了 ASCII 编码.

UTF-16 用二或四个字节来代表一个 Unicode code point.
其在包含中文, 日文或韩文等的场景中, 可能比 UTF-8 更高效.

## rune
Go 中的 [rune](https://go.dev/ref/spec#Rune_literals) 是指 int32, 对应 Unicode 字符集中的 code point.
其值包括在单引号中，既可以是字符, 如 `'a'`, 也可以是由反斜杠转义的内容, 如
- 反斜杠加单个字符代表一些特殊值, 包括 `\n`, `\t`, `\'`, `\\` 等
- \x 跟两个十六进制数字, 如 `\x41` 代表 A
- \u 跟四个十六进制数字, 如 `\u0041` 代表 A
- \U 跟八个十六进制数字, 如 `\U00000041` 代表 A
- \ 跟三个8进制数字, 如 `\x101` 代表 a

```go
func ExampleRune() {
	r := 'A'
	fmt.Println(r == '\x41')
	fmt.Println(r == '\u0041')
	fmt.Println(r == '\U00000041')
	fmt.Println(r == '\101')
	// Output: true
	// true
	// true
	// true
}
```

## string
Go 中的 string 有两种形式: raw 和 interpreted.
前者用反引号\`, 如 \`foo\`, 后者用双引号, 如 "foo".

我们可以在 interpreted literal string 中使用 rune 的反斜杠转义来表示任意字符, 如:
```go
func ExampleLiteralString() {
	fmt.Print("Hello\n")
	fmt.Print("H\x65llo\n")
	fmt.Print("H\u0065llo\n")
	fmt.Print("H\145llo\n")
	// Output: Hello
	// Hello
	// Hello
	// Hello
}
```

我们并不能在在 interpreted literal string 中转义反引号, "\`" 不是一个合法的字符串,
因为反引号在 Go 中用于表示 raw literal string.
不同于 interpreted string, raw string 并不处理任何转义.
```go
func ExampleRawString() {
	fmt.Println(`H\x64llo`)
	// Output: H\x64llo
}
```

Go 中的 string 实质是只读的字节数组(read-only slice of bytes).
string literal 按 UTF-8 编码后存放在字节数组中, 如 "中国" 就需要占用 6 个字节.
```go
func ExampleStringIsBytes() {
	fmt.Println(len("中国"))
	fmt.Println(string([]byte{0xe4, 0xb8, 0xad, 0xe5, 0x9b, 0xbd}))
	// Output: 6
	// 中国
}
```

这种情况下, 遍历字符串是逐字节遍历字节数组而不是逐字符遍历字符串.
大多数人对此感知不明显的主要原因是:
- 大部分时候处理的都是 ASCII 字符, 仅占一字节
- 诸如 for-range 等语法糖是按字符遍历
```
func ExampleIterString() {
	s := "中国"
	for i := 0; i < len(s); i++ {
		fmt.Printf("%x,", s[i])
	}
	fmt.Println("")
	for _, c := range s {
		fmt.Printf("%c,", c)
	}
	fmt.Println("")
	// Output: e4,b8,ad,e5,9b,bd,
	// 中,国,
}
```

## Reference
- [Strings, bytes, runes and characters in Go](https://go.dev/blog/strings)
- [Rune literals](https://go.dev/ref/spec#Rune_literals)
- [String literals](https://go.dev/ref/spec#String_literals)
