package main

func add(a, b int) int {
	return a + b
}

func main() {
	f := func(a, b int) int {
		return a + b
	}

	f(1, 2)
	add(1, 2)
}
