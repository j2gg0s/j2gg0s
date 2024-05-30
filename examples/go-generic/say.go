package main

//go:noinline
func Say[T interface {
	LastName() string
	Name() string
}](v T) string {
	if v.LastName() != "unknown" {
		return v.LastName() + v.Name() + " say hello"
	}
	return v.Name() + " say hello"
}

type Foo struct{}

func (User) LastName() string { return "unknown" }

//go:noinline
func (User) Name() string { return "foo" }

type Bar struct{}

func (Bar) LastName() string { return "unknown" }

//go:noinline
func (Bar) Name() string { return "bar" }

//go:noinline
func FooSay(v User) string {
	return v.Name() + " say hello"
}

func main() {
	foo, bar := User{}, Bar{}

	Say[User](foo)
	FooSay(foo)

	Say[Bar](bar)
}
