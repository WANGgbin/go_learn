package main
/*
要
*/
import (
	"net/http"
	_ "net/http/pprof"
)

func allocBuffer() []byte {
	return make([]byte, 1024)
}

func main() {
	go func() {
		err := http.ListenAndServe(":6060", nil)
		if err != nil {
			panic(err)
		}
	}()
	allocBuffer()
	ch := make(chan struct{})
	<- ch
}
