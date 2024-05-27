package main

type Mather interface {
	Add(a, b int32) int32
	Sub(a, b int64) int64
}

type Inter struct{ id int32 }

//go:noinline
func (adder Inter) Add(a, b int32) int32 { return a + b }

//go:noinline
func (adder Inter) Sub(a, b int64) int64 { return a - b }

type Structer struct {
	x int32
	y string
}

//go:noinline
func (adder Structer) Add(a, b int32) int32 { return a + b }

//go:noinline
func (adder Structer) Sub(a, b int64) int64 { return a - b }

func main() {
	// NOTE: avoid devirtualizing
	for _, m := range []Mather{Mather(Inter{id: 6754}), Mather(Structer{7, "8"})} {
		m.Add(10, 32)
	}
	any
}
