package main

func add(ch chan int) int {
	a := <-ch
	b := <-ch
	return a + b
}

func main() {
	ch := make(chan int, 2)
	defer close(ch)
	ch <- 10
	ch <- 20
	add(ch)
}
