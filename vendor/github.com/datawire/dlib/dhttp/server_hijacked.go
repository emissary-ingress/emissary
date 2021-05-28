package dhttp

import (
	"context"
	"net"
	"net/http"
	"sync"
)

type connContextKey struct{}

// configureHijackTracking configures (mutates) an *http.Server to provide slightly better tracking
// of Hijack()ed connections.
//
// Hijack()ed connections are connections for which the http.Server.Handler said to the *http.Server
// "you know what, stop doing HTTP and let me use the raw TCP socket" (as when having negotiated an
// upgrade from HTTP/1 to WebSockets or from HTTP/1 to HTTP/2).  It is a design deficiency in
// *http.Server that the *http.Server totally just stops tracking connections when they get
// Hijack()ed; failing to close them when you call *http.Server.Close() and failing to wait for them
// when you call *http.Server.Shutdown(); and unfortunately that design deficiency is locked-in to
// *http.Server because of backward compatibility promises.
//
// So configureHijackTracking addresses that deficiency, and adds hooks in to the *http.Server in
// order to do its own tracking of the Hijack()ed connections, and returns two functions that will
// perform those two deficient operations for Hijack()ed connections.  It returns a 'close' function
// that closes all active Hijack()ed connections (you should call this when you call server.Close),
// and a 'wait' function that blocks until all of the workers have quit (you should call this
// immediately after you call server.Shutdown).
//
// This wraps the server.Handler, so it should be called *after* setting up any Handler that might
// Hijack() connections.
func configureHijackTracking(server *http.Server) (close func(), wait func()) {
	var wg sync.WaitGroup

	var mu sync.Mutex                            // protects 'hijackedConns'
	hijackedConns := make(map[net.Conn]struct{}) // protected by 'mu'
	closeHijacked := func() {
		mu.Lock()
		defer mu.Unlock()
		for conn := range hijackedConns {
			conn.Close()
			delete(hijackedConns, conn)
		}
	}

	// Hook in to .ConnState in order to make a note of it whenever a connection gets hijacked.
	origConnState := server.ConnState
	server.ConnState = func(conn net.Conn, state http.ConnState) {
		if origConnState != nil {
			origConnState(conn, state)
		}
		if state == http.StateHijacked {
			mu.Lock()
			hijackedConns[conn] = struct{}{}
			mu.Unlock()
		}
	}

	// Hook in to .ConnContext in order to pack the net.Conn in to the Context, so that we can
	// access it below.
	server.ConnContext = concatConnContext(
		func(ctx context.Context, c net.Conn) context.Context {
			return context.WithValue(ctx, connContextKey{}, c)
		},
		server.ConnContext,
	)

	// Hook in to .Handler in order to (1) make a note of it whenever a hijacked connection's
	// worker returns, so that we don't need to keep track of that connection forever; and (2)
	// keep track of whether there are still outstanding workers.
	origHandler := server.Handler
	if origHandler == nil {
		origHandler = http.DefaultServeMux
	}
	server.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		wg.Add(1)
		defer wg.Done()
		defer func() {
			mu.Lock()
			defer mu.Unlock()
			conn := r.Context().Value(connContextKey{}).(net.Conn)
			delete(hijackedConns, conn)
		}()
		origHandler.ServeHTTP(w, r)
	})

	// Return.
	return closeHijacked, wg.Wait
}
