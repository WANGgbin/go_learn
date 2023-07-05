package main

import "net/http"

func main() {
	http.HandleFunc("/ping", func(w http.ResponseWriter, req *http.Request){
		_, _ = w.Write([]byte("pong"))
	})
	http.ListenAndServe(":8080", nil)
}