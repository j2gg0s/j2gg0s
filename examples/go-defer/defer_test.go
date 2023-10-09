package main

import (
	"sync"
	"testing"
)

var mu sync.Mutex

//go:noinline
func deferLock() {
	mu.Lock()
	defer mu.Unlock()
}

//go:noinline
func noDeferLock() {
	mu.Lock()
	mu.Unlock()
}

func BenchmarkDeferLock(b *testing.B) {
	for i := 0; i < b.N; i++ {
		deferLock()
	}
}

func BenchmarkNoDeferLock(b *testing.B) {
	for i := 0; i < b.N; i++ {
		noDeferLock()
	}
}
