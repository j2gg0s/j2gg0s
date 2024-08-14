package main

import (
	"fmt"
	"reflect"
)

func main() {
	i := 0
	fmt.Println(reflect.ValueOf(i).CanSet())         // false: 值传递, 拷贝, 修改无意义, 所以结果为 false
	fmt.Println(reflect.ValueOf(&i).CanSet())        // false: 注意此时参数是指向 i 的指针, 修改这个指针依然是没有意义的
	fmt.Println(reflect.ValueOf(&i).Elem().CanSet()) // true: 如果我们想修改 i, 则需要先获取指针指向的元素

	// reflect.ValueOf(i).Set(1)	遵循 Go 的哲学, 在遇到此类无意义的操作时, 直接 panic, 而不是沉默

	reflect.ValueOf(&i).Elem().SetInt(1)
	fmt.Println(i) // 1
}

// [The Laws of Reflection](https://go.dev/blog/laws-of-reflection)
// 1. Reflection goes from interface value to reflection object.
// 2. Reflection goes from reflection object to interface value.
// 3. To modify a reflection object, the value must be settable.
//
// 对于第一条, Go 会自动将参数转为 any.
