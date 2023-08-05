package main

type Person struct {
	age int
}

func (p *Person) SetAge(param int) {
	p.age = param
}

type Student struct {
	Person
}


func main() {
	var s []int
	s1 := s[:0]
	println(s1)
}
