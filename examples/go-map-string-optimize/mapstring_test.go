package main

import (
	"testing"
)

func BenchmarkMapStringWithString(b *testing.B) {
	for i := 0; i < b.N; i++ {
		getByString(m, key)
	}
}

func BenchmarkMapStringWithBytes(b *testing.B) {
	for i := 0; i < b.N; i++ {
		getByBytes(m, key)
	}
}
