package main

//go:noinline
func Sum[T interface{ ~int | ~int32 | ~int64 }](nums []T) (s T) {
	if len(nums) == 0 {
		return
	}
	for _, n := range nums {
		s += n
	}
	return
}

func main() {
	Sum([]int{1, 2, 3})
}
