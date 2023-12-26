package main

import (
	"sync/atomic"
	"time"
)

var x atomic.Int64

func load() int64 {
	return x.Load()
}

func store(i int) {
	x.Store(int64(i))
}

func main() {
	go store(32)
	go load()
	time.Sleep(1)
}
