package main

import "time"

func main() {
	t := time.Date(1998, time.June, 8, 0, 0, 0, 999999999, time.Local)
	print(t.Format(time.RFC3339Nano))
}