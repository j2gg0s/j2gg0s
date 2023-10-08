package main

//go:noinline
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func main() {
	max(10, 20)

	imax := max
	imax(10, 20)

	x := 1
	y := 2
	iadd := func(a, b int) int {
		return x + y + a + b
	}
	iadd(10, 20)

	// 直接调用并不需要特殊实现, 即使是闭包
	func(a, b int) int {
		return x + y + a + b
	}(10, 20)
}
