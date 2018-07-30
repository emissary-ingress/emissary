package main

import (
	"fmt"
	"log"
	"net/http"
	"encoding/json"
	"strconv"
)

func requestLogger(w http.ResponseWriter, r *http.Request) {
	var request = make(map[string]interface{})
	request["url"] = r.URL
	request["method"] = r.Method
	request["header"] = r.Header

	// respond with the requested status
	status := r.Header.Get("Requested-Status")
	if status == "" {
		status = "200"
	}
	
	statusCode, err := strconv.Atoi(status)
	if err != nil {
		log.Print(err)
		statusCode = 500
	}

	// copy the requested headers into the response
	headers, ok := r.Header["Requested-Header"]
	if ok {
		for _, header := range headers {
			canonical := http.CanonicalHeaderKey(header)
			value, ok := r.Header[canonical]
			if ok {
				w.Header()[canonical] = value
			}
		}
	}

	w.WriteHeader(statusCode)

	// Write out all request/response information
	var response = make(map[string]interface{})
	response["headers"] = w.Header()

	var body = make(map[string]interface{})
	body["request"] = request
	body["response"] = response

	b, err := json.MarshalIndent(body, "", "  ")
	if err != nil {
		b = []byte(fmt.Sprintf("Error: %v", err))
	}

	w.Write(b)
}

func main() {
	http.HandleFunc("/", requestLogger)
	log.Fatal(http.ListenAndServe(":8080", nil))
}
