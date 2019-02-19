package util

import (
	"encoding/json"
	"net/http"
	"net/url"
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

// OriginalURL(r) is like r.URL, but obeys `Host` and
// `X-Forwarded-Proto`.
//
// TODO(lukeshu): Use RFC 7239 `Forwarded` instead of the old
// non-standard `X-Forwarded-Proto`.
func OriginalURL(r *http.Request) *url.URL {
	u, _ := r.URL.Parse("")
	u.Host = r.Host
	if r.TLS != nil || r.Header.Get("x-forwarded-proto") == "https" {
		u.Scheme = "https"
	} else {
		u.Scheme = "http"
	}
	return u
}
