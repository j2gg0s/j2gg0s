package main

import (
	"fmt"
	"io"
	"testing"
)

func BenchmarkInterfaceAllocate(b *testing.B) {
	u := User{"Foo", 35}

	var v interface{}
	for i := 0; i < b.N; i++ {
		v = interface{}(u)
	}

	fmt.Fprintln(io.Discard, v)
}

func BenchmarkInterfaceNoAllocate(b *testing.B) {
	var v interface{}
	for i := 0; i < b.N; i++ {
		v = interface{}(32)
	}

	fmt.Fprintln(io.Discard, v)
}

func BenchmarkInterfacePtr(b *testing.B) {
	u := &User{"Foo", 35}

	var v interface{}
	for i := 0; i < b.N; i++ {
		v = interface{}(u)
	}

	fmt.Fprintln(io.Discard, v)
}
