package main

import (
	"fmt"
	"math/rand"
	"net/http"
	_ "net/http/pprof"
	"syscall"
	"time"
	_ "unsafe"
)

const (
	BunkSize    = 4096
	ThreadCount = 9900
)

var FcntlSyscall uintptr = syscall.SYS_FCNTL

func isNonBlock(fd int) (bool, error) {
	flag, _, e1 := syscall.Syscall(FcntlSyscall, uintptr(fd), uintptr(syscall.F_GETFL), 0)
	if e1 != 0 {
		return false, e1
	}
	return flag&syscall.O_NONBLOCK != 0, nil
}

func readFromStdIn(ch chan<- struct{}) {
	buf := make([]byte, BunkSize)
	isNonBlock, err := isNonBlock(syscall.Stdin)
	if err != nil {
		panic(err)
	}
	if isNonBlock {
		panic(fmt.Sprintf("fd: %d is not block", syscall.Stdin))
	}
	time.Sleep(time.Duration(rand.Intn(20)) * time.Second)
	count, err := syscall.Read(syscall.Stdin, buf)
	if err != nil {
		fmt.Printf("error: %v\n", err)
	}
	fmt.Printf("read %d bytes\n", count)
	ch <- struct{}{}
}
func main() {
	go func() {
		http.ListenAndServe(":6060", nil)
	}()
	//syscall.SetNonblock(syscall.Stdin, true)
	ch := make(chan struct{}, ThreadCount)
	// time.Sleep(5 * time.Second)
	for i := 0; i < ThreadCount; i++ {
		go readFromStdIn(ch)
	}
	for i := 0; i < ThreadCount+1; i++ {
		<-ch
	}
}
