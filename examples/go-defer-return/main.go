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
