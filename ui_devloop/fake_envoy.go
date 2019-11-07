package main

import (
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
)

func main() {
	target, err := url.Parse("http://localhost:8501/edge_stack_ui")
	if err != nil {
		panic(err)
	}
	log.Fatal(http.ListenAndServe(":8081", httputil.NewSingleHostReverseProxy(target)))
}
