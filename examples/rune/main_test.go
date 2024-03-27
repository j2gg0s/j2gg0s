package main

import (
	"fmt"
)

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

func ExampleRawString() {
	fmt.Println(`H\x64llo`)
	// Output: H\x64llo
}

func ExampleStringIsBytes() {
	fmt.Println(len("中国"))
	fmt.Println(string([]byte{0xe4, 0xb8, 0xad, 0xe5, 0x9b, 0xbd}))
	// Output: 6
	// 中国
}

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
