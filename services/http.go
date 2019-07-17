package services

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

// HTTP server object (all fields are required).
type HTTP struct {
	Port          int16
	Backend       string
	SecurePort    int16
	SecureBackend string
	Cert          string
	Key           string
}

// Start initializes the HTTP server.
func (h *HTTP) Start() <-chan bool {
	log.Printf("HTTP: %s listening on %d/%d", h.Backend, h.Port, h.SecurePort)

	server := http.NewServeMux()
	server.HandleFunc("/", h.handler)

	exited := make(chan bool)

	go func() {
		log.Fatal(http.ListenAndServe(fmt.Sprintf(":%v", h.Port), server))
		close(exited)
	}()

	go func() {
		log.Fatal(http.ListenAndServeTLS(fmt.Sprintf(":%v", h.SecurePort), h.Cert, h.Key, server))
		close(exited)
	}()

	return exited
}

// Helpers
func lower(m map[string][]string) (result map[string][]string) {
	result = make(map[string][]string)
	for k, v := range m {
		result[strings.ToLower(k)] = v
	}
	return result
}

func (h *HTTP) handler(w http.ResponseWriter, r *http.Request) {
	// Assume we're the clear side of the world.
	backend := h.Backend
	conntype := "CLR"

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
		if ok {
			url["password"] = pw
		}
	}

	request["method"] = r.Method
	request["headers"] = lower(r.Header)
	request["host"] = r.Host
	var tls = make(map[string]interface{})
	request["tls"] = tls

	tls["enabled"] = r.TLS != nil

	if r.TLS != nil {
		// We're the secure side of the world, I guess.
		backend = h.SecureBackend
		conntype = "TLS"

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

	if b, _ := ioutil.ReadAll(r.Body); b != nil {
		body := string(b)
		log.Printf("received body: %s", body)
		w.Header()[http.CanonicalHeaderKey("Auth-Request-Body")] = []string{body}
	}
	defer r.Body.Close()

	cookies, ok := r.Header["Requested-Cookie"]
	if ok {
		for _, v := range strings.Split(cookies[0], ",") {
			val := strings.Trim(v, " ")
			http.SetCookie(w, &http.Cookie{
				Name:  val,
				Value: val,
			})
		}
	}

	// If they asked for a specific location to be returned, handle that too.
	location, ok := r.Header["Requested-Location"]

	if ok {
		w.Header()[http.CanonicalHeaderKey("Location")] = location
	}

	addExtauth := os.Getenv("INCLUDE_EXTAUTH_HEADER")

	if len(addExtauth) > 0 {
		extauth := make(map[string]interface{})
		extauth["request"] = request
		extauth["resp_headers"] = lower(w.Header())

		eaJSON, err := json.Marshal(extauth)

		if err != nil {
			eaJSON = []byte(fmt.Sprintf("err: %v", err))
		}

		eaArray := make([]string, 1, 1)
		eaArray[0] = string(eaJSON)

		w.Header()[http.CanonicalHeaderKey("extauth")] = eaArray
	}

	// Check header and delay response.
	if h, ok := r.Header["Requested-Backend-Delay"]; ok {
		if v, err := strconv.Atoi(h[0]); err == nil {
			log.Printf("Delaying response by %v ms", v)
			time.Sleep(time.Duration(v) * time.Millisecond)
		}
	}

	// Set date response header.
	w.Header().Set("Date", time.Now().Format("Wed, 17 Jul 2019 15:43:03 GMT"))

	w.WriteHeader(statusCode)

	// Write out all request/response information
	var response = make(map[string]interface{})
	response["headers"] = lower(w.Header())

	var body = make(map[string]interface{})
	body["backend"] = backend
	body["request"] = request
	body["response"] = response

	b, err := json.MarshalIndent(body, "", "  ")
	if err != nil {
		b = []byte(fmt.Sprintf("Error: %v", err))
	}

	log.Printf("%s (%s): writing response HTTP %v", backend, conntype, statusCode)
	w.Write(b)
}
