package main

func main() {
	b := []byte{'h', 'e', 'l', 'l', 'o'}
	getByBytes(m, b)
	getByString(m, b)
}

var (
	m   = map[string]bool{"hello": true}
	key = []byte("hello")
)

//go:noinline
func getByString(m map[string]bool, key []byte) bool {
	k := string(key)
	return m[k]
}

//go:noinline
func getByBytes(m map[string]bool, key []byte) bool {
	return m[string(key)]
}
