package server

import (
	"net/http"

	"github.com/lyft/goruntime/loader"
	stats "github.com/lyft/gostats"
	"google.golang.org/grpc"
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
	 * all endpoints have been registered with the
	 * DebugHTTPHandler() and the GrpcServer().
	 */
	Start()

	/**
	 * Returns the root of the stats tree for the server
	 */
	Scope() stats.Scope

	/**
	 * Returns the embedded HTTP handler to be used for debugging.
	 */
	DebugHTTPHandler() DebugHTTPHandler

	/**
	 * Returns the embedded gRPC server to be used for registering gRPC endpoints.
	 */
	GrpcServer() *grpc.Server

	/**
	 * Returns the runtime configuration for the server.
	 */
	Runtime() loader.IFace
}
