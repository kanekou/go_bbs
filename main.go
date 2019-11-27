package main

import (
	"fmt"
	"net/http"
)

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("/", handler)
	s := &http.Server{
		Addr:    ":9000",
		Handler: mux,
	}
	s.ListenAndServe()
}

func handler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, "Hello, World")
}
