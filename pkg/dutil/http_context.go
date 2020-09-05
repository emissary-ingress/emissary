package dutil

import (
	"context"
	"net"
	"net/http"

	"github.com/datawire/ambassador/pkg/dcontext"
)

// ListenAndServeHTTPWithContext runs server.ListenAndServe() on an
// http.Server, but properly calls server.Shutdown when the Context is
// canceled.
//
// It obeys hard/soft cancellation as implemented by
// dcontext.WithSoftness; it calls server.Shutdown() when the soft
// Context is canceled, and the hard Context being canceled causes the
// .Shutdown() to hurry along and kill any live requests and return,
// instead of waiting for them to be completed gracefully.
func ListenAndServeHTTPWithContext(ctx context.Context, server *http.Server) error {
	server.BaseContext = func(_ net.Listener) context.Context { return dcontext.HardContext(ctx) }
	serverCh := make(chan error)
	go func() {
		serverCh <- server.ListenAndServe()
	}()
	select {
	case err := <-serverCh:
		return err
	case <-ctx.Done():
		return server.Shutdown(dcontext.HardContext(ctx))
	}
}
