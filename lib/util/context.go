package util

import (
	"context"
	"net/http"
)

// ListenAndServeHTTPWithContext runs server.ListenAndServe() on an
// http.Server(), but properly calls server.Shutdown when the context
// is canceled.
//
// softCtx should be a child context of hardCtx.  softCtx being
// canceled triggers server.Shutdown().  If hardCtx being cacneled
// triggers that .Shutdown() to kill any live requests and return,
// instead of waiting for them to be completed gracefully.
func ListenAndServeHTTPWithContext(hardCtx, softCtx context.Context, server *http.Server) error {
	serverCh := make(chan error)
	go func() {
		serverCh <- server.ListenAndServe()
	}()
	select {
	case err := <-serverCh:
		return err
	case <-softCtx.Done():
		return server.Shutdown(hardCtx)
	}
}
