package main

import "fmt"

type struct1 struct {
	i int
	p *int
}

type struct2 struct {
	i int
	p *float32
}

func print[T any](v T) {
	fmt.Print(v)
}

func add[T int | float32](i, j T) T {
	return i + j
}

func add[T float64](i, j T) T {
	return i + j
}

func main() {
	fmt.Print(add(1, 2))
}
