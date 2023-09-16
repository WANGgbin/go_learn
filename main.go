package main

type Person struct {
	Age int
}

func (p *Person) SetAge(param int) {
	p.Age = param
}

type Student struct {
	Person
}

type myInt  int
func (m *myInt) set(p int) {
	*m = myInt(p)
}

func main() {
	//reflect.ValueOf().Call()
	//reflect.ValueOf().Method().Call()
	//reflect.TypeOf().Method()
	//reflect.TypeOf().Implements()
	//reflect.MakeFunc()
	//reflect.MakeSlice()
	//reflect.StructOf()
	//errors.Is()
	s1 := make([]int, 3)
	s1[0] = 0
	s1[1] = 1
	s1[2] = 2

	s2 := make([]int, 4)
	print(copy(s2, s1))

}

