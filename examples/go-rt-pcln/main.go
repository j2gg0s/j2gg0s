package main

//go:noinline
func add(a, b int) (int, bool) {
	return a + b, true
}

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

func main() {
	add(10, 20)
	max(10, 20)
	sum([]int{10, 20})
}
