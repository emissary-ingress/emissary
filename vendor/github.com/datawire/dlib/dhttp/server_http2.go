package dhttp

import (
	"net/http"

	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
)

// configureHTTP2 configures (mutates) an *http.Server to handle HTTP/2 connections, including both
// "h2" (encrypted HTTP/2) and "h2c" (cleartext HTTP/2) connections.  If the Server is not run with
// TLS, then encrypted "h2" will effectively be disabled.
//
// The HTTP/2 configuration 'conf' may be nil.
//
//  - This is better than golang.org/x/net/http2.ConfigureServer because this also handles "h2c",
//    not just "h2".
//  - This is better than the default net/http.Server HTTP/2 support, as the default support is
//    exactly equivalent to golang.org/x/net/http2.ConfigureServer (it uses a vendor'ed copy of
//    golang.org/x/net/http2).
//  - This is better than golang.org/x/net/http2/h2c.NewHandler because this will properly shut down
//    idle h2c connections when server.Shutdown is called, rather than allowing h2c connections to
//    sit around forever.
//
// This must be called before server starts serving, as this will mutate server.TLSConfig,
// server.TLSNextProto, and server.Handler.
//
// However, this has some limitations (that I believe all other alternatives also share):
//  - h2c connections are not closed by server.Close().
//  - server.Shutdown() may return early before all h2c connections have been shutdown.
// These limitations can be solved with configureHijackTracking.
func configureHTTP2(server *http.Server, conf *http2.Server) error {
	if server == nil {
		// This check mimics http2.ConfigureServer.  We explicitly check for it here
		// (instead of just letting a nil-pointer panic implicitly be thrown when we access
		// server.Handler below) so that it's clear that this is intentional behavior.
		panic("nil *http.Server")
	}
	if conf == nil {
		// http2.ConfigureServer below will do this same check, but we need to do it early,
		// so that 'conf.state' can be shared between the "h2" and the "h2c" handlers.
		conf = new(http2.Server)
	}

	// Configure "h2c", except that this doesn't configure shutdown yet.  What we'll still need
	// to do for shutdown is call 'server.RegisterOnShutdown(conf.state.startGracefulShutdown)';
	// but we can't directly do that because neither 'state' nor 'startGracefulShutdown' are
	// exported.  Fortunately, the call to ConfigureServer (below) makes that call for us.
	// That'll work because they both use the same 'conf' and thus the same 'conf.state' (see
	// above).
	origHandler := server.Handler
	if origHandler == nil {
		origHandler = http.DefaultServeMux
	}
	server.Handler = h2c.NewHandler(origHandler, conf)

	// Configure "h2", this also configures shutdown for "h2c" (see above).
	return http2.ConfigureServer(server, conf)
}
