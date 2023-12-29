package main

import (
	"fmt"
	"time"
)

//go:noinline
func fnClosure(v int) {
	go func() {
		fmt.Println(v)
	}()
}

//go:noinline
func fnLoop(nums []int) {
	for i, v := range nums {
		fmt.Println(&i, &v)
	}
}

//go:noinline
func fnLoopClosure() {
	for i := 0; i < 10; i++ {
		go func() {
			fmt.Println(i)
		}()
	}
}

//go:noinline
func fnVarScope() {
	s := "hello world"
	{
		s := 10
		fmt.Println("s:", s)
	}
	fmt.Println("s:", s)
}

func main() {
	nums := []int{1, 2, 3}
	fnLoop(nums)

	fnClosure(10)

	fnLoopClosure()

	fnVarScope()

	time.Sleep(time.Second)
}
