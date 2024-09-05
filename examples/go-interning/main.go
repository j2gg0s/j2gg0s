package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"runtime"
	"runtime/pprof"
)

var (
	memprofile = flag.String("memprofile", "mem.prof", "")
	interning  = flag.Bool("interning", false, "")
)

func main() {
	flag.Parse()
	var words []string
	if *interning {
		words = readInterning()
	} else {
		words = read()
	}

	f, err := os.Create(*memprofile)
	if err != nil {
		panic(err)
	}
	defer f.Close()
	runtime.GC()
	if err := pprof.Lookup("heap").WriteTo(f, 0); err != nil {
		panic(err)
	}

	stats := &runtime.MemStats{}
	runtime.ReadMemStats(stats)
	fmt.Println("intering", *interning, "heapAlloc", stats.HeapAlloc, "words", len(words), words[len(words)-1])
}

func readInterning() []string {
	f, err := os.Open("1984.txt")
	if err != nil {
		panic(err)
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	scanner.Split(bufio.ScanWords)

	words := []string{}
	for scanner.Scan() {
		words = append(words, Intern(scanner.Bytes()))
	}
	if err := scanner.Err(); err != nil {
		panic(err)
	}

	return words
}

func read() []string {
	f, err := os.Open("1984.txt")
	if err != nil {
		panic(err)
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	scanner.Split(bufio.ScanWords)

	words := []string{}
	for scanner.Scan() {
		word := scanner.Text()
		words = append(words, word)
	}
	if err := scanner.Err(); err != nil {
		panic(err)
	}
	return words
}

var internPool = map[string]string{}

// Intern returns a string that is equal to s but that may share storage with
// a string previously passed to Intern.
func Intern(b []byte) string {
	s, ok := internPool[string(b)]
	if ok {
		return s
	}
	s = string(b)
	internPool[s] = s
	return s
}
