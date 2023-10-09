package main

//go:noinline
func max(a, b int) int {
	defer func() {}()
	defer func() {}()
	defer func() {}()
	if a > b {
		return a
	}
	return b
}

//go:noinline
func sum(numbers []int) int {
	sum := 0
	for i := 0; i < len(numbers); i++ {
		defer func() {}()
		sum += numbers[i]
	}
	return sum
}

//go:noinline
func add(a, b int) int {
	defer func() {}()
	defer func() {}()
	defer func() {}()
	defer func() {}()
	defer func() {}()
	defer func() {}()
	defer func() {}()
	defer func() {}()
	defer func() {}()
	return a + b
}

func main() {
	add(10, 20)
	max(10, 20)
	sum([]int{10, 20})
}
