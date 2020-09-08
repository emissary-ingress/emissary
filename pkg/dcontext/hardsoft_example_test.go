package dcontext_test

import (
	"context"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/datawire/ambassador/pkg/dcontext"
)

func ExampleWithSoftCancel(t *testing.T) {
	ctx := context.Background()                       // Context is hard by default
	ctx, timeToDie := context.WithCancel(ctx)         // hard Context => hard cancel
	ctx = dcontext.WithSoftness(ctx)                  // make it soft
	ctx, startShuttingDown := context.WithCancel(ctx) // soft Context => soft cancel

	go ListenAndServeHTTPWithContext(ctx, &http.Server{
		// ...
	})

	// run for a while
	time.Sleep(10 * time.Second)

	// shut down
	startShuttingDown()         // start draining connections
	time.Sleep(5 * time.Second) // grace period
	timeToDie()                 // if there are connections that are still in use after the 5-second grace period, kill them forcefully
}

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
	// Regardless of if you use dcontext, you should always set
	// `.BaseContext` on your `http.Server`s so that your HTTP Handler
	// receives a request object that has `Request.Context()` set correctly.
	server.BaseContext = func(_ net.Listener) context.Context {
		// We use the hard Context here instead of the soft Context so
		// that in-progress requests don't get interrupted when we enter
		// the shutdown grace period.
		return dcontext.HardContext(ctx)
	}
	serverCh := make(chan error)
	go func() {
		serverCh <- server.ListenAndServe()
	}()
	select {
	case err := <-serverCh:
		// The server quit on its own.
		return err
	case <-ctx.Done():
		// A soft shutdown has been initiated; call server.Shutdown().
		return server.Shutdown(dcontext.HardContext(ctx))
	}
}
