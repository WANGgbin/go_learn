package main

import (
	"encoding/json"
	"fmt"
)

func add(a, b int) int {
	return a + b
}

type Person struct {
	Self *Person
	Name string
	Age  int
}

func (p *Person) GetName() string {
	return p.Name
}

func (p *Person) GetAge() int {
	return p.Age
}

func main() {
	p := &Person{Name: "wgb", Age: 12}
	p.Self = p

	info, err := json.Marshal(p)
	if err != nil {
		panic(err)
	}
	fmt.Printf("%s:\n", string(info))
}
