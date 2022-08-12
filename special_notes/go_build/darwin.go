//go:build windows || darwin

package main

import "fmt"

func Print() {
	fmt.Printf("called windows's Print\n")
}
