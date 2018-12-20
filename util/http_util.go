package util

import (
	"bytes"
	"encoding/json"
	"net/http"
)

// Error is the default error response object ..
type Error struct {
	Message string `json:"message"`
}

// HTTPContextKey is a helper to define an HTTP request context key.
type HTTPContextKey string

func (c HTTPContextKey) String() string {
	return string(c)
}

// ToJSONResponse takes the HTTP response writer object, the status code, a json struct and
// sets the writer to produce a json response.
func ToJSONResponse(w http.ResponseWriter, status int, i interface{}) {
	jsonResponse, err := json.Marshal(i)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "Application/Json")
	w.WriteHeader(status)
	w.Write(jsonResponse)
}

// ToRawURL is a handy method for extracting the FQDN from the a client's request.
func ToRawURL(r *http.Request) string {
	var buf bytes.Buffer

	if r.TLS != nil {
		buf.WriteString("https://")
	} else {
		buf.WriteString("http://")
	}
	buf.WriteString(r.Host)
	buf.WriteString(r.RequestURI)

	return buf.String()
}
