package main

import (
	"net/http"
	"os"

	//nolint:depguard // Because of the simple Dockerfile, we are limited to stdlib.
	"log"
)

func main() {
	body := os.Getenv("HTTPTEST_BODY")
	if body == "" {
		body = "HTTPTEST"
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		_, err := w.Write([]byte(body))
		if err != nil {
			log.Print(err)
		}
	})

	log.Fatal(http.ListenAndServe(":8080", mux))
}
