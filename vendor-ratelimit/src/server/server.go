package server

import (
	"net/http"
)

type DebugHTTPHandler interface {
	http.Handler

	/**
	 * Add an HTTP endpoint to the local debug port.
	 */
	AddEndpoint(path string, help string, handler http.HandlerFunc)
}
