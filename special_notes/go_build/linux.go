//go:build linux || darwin
// +build linux darwin

package main

import "fmt"

func Print() {
	fmt.Printf("called linux's Print\n")
}
