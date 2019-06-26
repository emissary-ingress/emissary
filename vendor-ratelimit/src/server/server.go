package server

import (
	"net/http"

	"github.com/lyft/goruntime/loader"
	stats "github.com/lyft/gostats"
)

type DebugHTTPHandler interface {
	http.Handler

	/**
	 * Add an HTTP endpoint to the local debug port.
	 */
	AddEndpoint(path string, help string, handler http.HandlerFunc)
}

type Server interface {
	/**
	 * Starts the HTTP and gRPC servers. This should be done after
	 * all endpoints have been registered with the DebugHTTPHandler
	 * and grpc.Server that were passed to NewServer().
	 */
	Start()

	/**
	 * Returns the root of the stats tree for the server
	 */
	Scope() stats.Scope

	/**
	 * Returns the runtime configuration for the server.
	 */
	Runtime() loader.IFace
}
