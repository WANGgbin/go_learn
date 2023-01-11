package main

func func1(i, j int64) int64 {
	return i + j
}

func func2(i, j, k int64) int64 {
	return i + j + k
}

type I interface {
	setInt(i int)
	getInt() int
	print() string
}

type t struct {
	i int
}

func (p t) setInt(i int) {
	p.i = i
}

func (p t) getInt() int {
	return p.i
}

func (p t) print() string {
	return "xxx"
}

func getPrint() func(p t) string {
	return t.print
}

func main() {
	obj := getPrint()
	obj(t{})
}
