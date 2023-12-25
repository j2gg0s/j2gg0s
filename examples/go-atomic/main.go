package main

import (
	"fmt"
	"sync/atomic"
)

var x atomic.Int64

func main() {
	x.Store(32)
	x.Add(1)
	x.CompareAndSwap(33, 34)
	i := x.Load()
	fmt.Println(i)
}
