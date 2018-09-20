package util

import (
	"encoding/json"
	"net/http"
)

// Error is the default error response object ..
type Error struct {
	Message string `json:"message"`
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
