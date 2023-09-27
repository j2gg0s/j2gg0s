package main

import "fmt"

//go:noinline
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

//go:noinline
func sum(numbers []int) int {
	sum := 0
	for i := 0; i < len(numbers); i++ {
		sum += numbers[i]
	}
	return sum
}

//go:noinline
func add(a, b int) int {
	defer func() { fmt.Println(1) }()
	defer func() { fmt.Println(2) }()
	defer func() { fmt.Println(3) }()
	defer func() { fmt.Println(4) }()
	defer func() { fmt.Println(5) }()
	defer func() { fmt.Println(6) }()
	defer func() { fmt.Println(7) }()
	defer func() { fmt.Println(8) }()
	defer func() { fmt.Println(9) }()
	return a + b
}

func main() {
	add(10, 20)
	max(10, 20)
}
