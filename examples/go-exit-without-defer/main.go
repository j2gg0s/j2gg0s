package main

import (
	"flag"
	"fmt"
	"os"
)

func main() {
	runPanic := flag.Bool("panic", true, "run panic")

	flag.Parse()

	if *runPanic {
		fnPanic()
	} else {
		fnExit()
	}
}

func fnPanic() {
	defer fn()
	fmt.Println("panic will trigger defer be executed")
	panic(nil)
}

func fnExit() {
	defer fn()
	fmt.Println("os.Exit will not trigger defer be executed")
	os.Exit(0)
}

func fn() {
	fmt.Println("defer be executed")
}

func fnPanicS() {
	defer func() {}()
	panic(nil)
}

func fnExitS() {
	defer func() {}()
	os.Exit(0)
}
