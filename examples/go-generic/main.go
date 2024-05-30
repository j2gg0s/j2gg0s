package main

type Sayer interface {
	Name() string
}

type SayerWithLastName interface {
	Name() string
	LastName() string
}

//go:noinline
func Say[T Sayer](v T) string {
	return v.Name() + " say hello"
}

type User struct {
	name     string
	lastName string
}

func (v *User) LastName() string { return v.lastName }

//go:noinline
func (v *User) Name() string { return v.name }

//go:noinline
func SayerSay(v Sayer) string {
	return v.Name() + " say hello"
}

func main() {
	foo := &User{name: "foo", lastName: "unknown"}

	SayerSay(foo)
	Say(foo)

	{
		i := SayerWithLastName(foo)
		Say(i)
	}
	{
		i := Sayer(foo)
		Say(i)
	}

	Say(&Anonymous{})
}

type Anonymous struct{}

func (v *Anonymous) Name() string { return "unknown" }
