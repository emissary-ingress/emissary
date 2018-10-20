package main

import (
	"fmt"
	"log"
	"net/http"
	"encoding/json"
	"os"
	"strconv"
	"strings"
)

func lower(m map[string][]string) (result map[string][]string) {
	result = make(map[string][]string)
	for k, v := range(m) {
		result[strings.ToLower(k)] = v
	}
	return result
}

func requestLogger(w http.ResponseWriter, r *http.Request) {
	var request = make(map[string]interface{})
	var url = make(map[string]interface{})
	request["url"] = url
	url["fragment"] = r.URL.Fragment
	url["host"] = r.URL.Host
	url["opaque"] = r.URL.Opaque
	url["path"] = r.URL.Path
	url["query"] = r.URL.Query()
	url["rawQuery"] = r.URL.RawQuery
	url["scheme"] = r.URL.Scheme
	if r.URL.User != nil {
		url["username"] = r.URL.User.Username()
		pw, ok := r.URL.User.Password()
		if ok { url["password"] = pw }
	}

	request["method"] = r.Method
	request["headers"] = lower(r.Header)
	request["host"] = r.Host
	var tls = make(map[string]interface{})
	request["tls"] = tls
	tls["enabled"] = r.TLS != nil
	if r.TLS != nil {
		tls["version"] = r.TLS.Version
		tls["negotiated-protocol"] = r.TLS.NegotiatedProtocol
		tls["server-name"] = r.TLS.ServerName
	}

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
	response["headers"] = lower(w.Header())

	var body = make(map[string]interface{})
	body["backend"] = os.Getenv("BACKEND")
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
	exited := make(chan bool)
	go func() {
		log.Fatal(http.ListenAndServe(":8080", nil))
		close(exited)
	}()
	go func() {
		log.Fatal(http.ListenAndServeTLS(":8443", "server.crt", "server.key", nil))
		close(exited)
	}()
	<- exited
}
