package main

func func1(i, j int64) int64 {
	return i + j
}

func func2(i, j, k int64) int64 {
	return i + j + k
}

func main() {
	i := int64(0)
	for {
		// time.Sleep(1 * time.Microsecond)
		i = int64(i + 1)
	}
}
