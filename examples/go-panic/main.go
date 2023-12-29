package main

import "fmt"

//go:noinline
func doPanic() {
	panic(nil)
}

func main() {
	a := 10
	b := 20
	c := 30

	if !false {
		fmt.Println(fmt.Sprintf("%t", a))
	}

	defer func() {
		fmt.Println(a)
	}()
	defer func() {
		fmt.Println(b)
	}()
	defer func() {
		r := recover()
		if r != nil {
			fmt.Println(r)
		}
		fmt.Println(c)
	}()
	doPanic()

}
