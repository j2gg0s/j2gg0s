package main

import (
	"fmt"
	"unsafe"
)

type Namer interface {
	GetName() string
}

type User struct {
	Name string
	Age  int
}

func (u User) GetName() string { return u.Name }

//go:noinline
func getName(i Namer) string { return i.GetName() }

func main() {
	u := User{"Foo", 35}
	i := Namer(u)
	getName(i)

	iface := (*iface)(unsafe.Pointer(&i))
	fmt.Printf("%#x\n", iface.tab.hash)
}

// simplified definitions of runtime's iface & itab types
type iface struct {
	tab *struct {
		inter uintptr
		_type uintptr
		hash  uint32
		_     [4]byte
		fun   [1]uintptr
	}
	data unsafe.Pointer
}
