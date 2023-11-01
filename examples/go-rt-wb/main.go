package main

var sink *int

func main() {
	foo := []int{1, 2, 3}
	sink = &foo[1]
}
