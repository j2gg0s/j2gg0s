package main

//go:noinline
func fnSelect(r1, r2, s chan int) (int, bool) {
	var i, z int
	var ok bool
	select {
	case i, ok = <-r1:
		z = 1 + i
	case i, ok = <-r2:
		z = 2 + i
	case s <- 0:
		z, ok = 0, true
	default:
	}
	return z, ok
}

func main() {
	r1, r2, s := make(chan int), make(chan int), make(chan int)
	fnSelect(r1, r2, s)
	close(r1)
	close(r2)
	close(s)
}
