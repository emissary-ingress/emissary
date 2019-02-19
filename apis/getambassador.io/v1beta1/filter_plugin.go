package v1

import (
	"net/http"
)

type FilterPlugin struct {
	Name    string       `json:"name"`
	Handler http.Handler `json:"-"`
}
