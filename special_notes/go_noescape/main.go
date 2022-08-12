package main

// go:noescape
func refLocalVar() *int64

func main() {
	ptr := refLocalVar()
	*ptr = 1
}
