package main

//go:inline
func add(x, y int) int { return x + y }

//go:noinline
func main() {
	add(1, 2)
}
