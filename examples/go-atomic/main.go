package main

import (
	"fmt"
	"sync/atomic"
	"time"
)

var x atomic.Int64
var y int64

func fn() {
	for x.Load() != 32 {
	}
	fmt.Println(y)
}

func main() {
	go fn()
	y = 33
	x.Store(32)

	ex001()
	time.Sleep(time.Second)
}
