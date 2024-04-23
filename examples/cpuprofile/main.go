package main

import (
	"fmt"
	"os"

	"github.com/google/pprof/profile"
)

func main() {
	r, err := os.Open("cpu.profile")
	if err != nil {
		panic(err)
	}

	p, err := profile.Parse(r)
	if err != nil {
		panic(err)
	}

	fmt.Println(p)
}
