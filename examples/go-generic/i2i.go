package main

type Namer interface {
	GetName() string
}

type NamerAndAger interface {
	GetName() string
	GetAge() int
}

type User struct {
	Name string
	Age  int
}

func (u User) GetName() string { return u.Name }
func (u User) GetAge() int     { return u.Age }

//go:noinline
func getName(i Namer) string { return i.GetName() }

func main() {
	u := NamerAndAger(User{"Foo", 35})
	getName(u)
}
