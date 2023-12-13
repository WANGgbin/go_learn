package main

import "testing"

func TestAdd(t *testing.T) {
	t.Log("begin...")
	expect := 3
	got := Add(1, 2)

	if got != expect {
		t.Fatalf("want %d, got %d", expect, got)
	}
	t.Log("end...")
}

func TestMain(m *testing.M) {
	m.Run()
	return
}
