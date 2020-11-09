package dcontext_test

import (
	"context"
	"net"
	"net/http"
	"time"

	"github.com/datawire/ambassador/pkg/dcontext"
)

// This should be a very simple example of a parent caller function, showing how
// to manage a hard/soft Context and how to call code that is dcontext-aware.
func Example_caller() error {
	ctx := context.Background()                       // Context is hard by default
	ctx, timeToDie := context.WithCancel(ctx)         // hard Context => hard cancel
	ctx = dcontext.WithSoftness(ctx)                  // make it soft
	ctx, startShuttingDown := context.WithCancel(ctx) // soft Context => soft cancel

	retCh := make(chan error)
	go func() {
		retCh <- ListenAndServeHTTPWithContext(ctx, &http.Server{
			// ...
		})
	}()

	// Run for a while.
	time.Sleep(10 * time.Second)

	// Shut down.
	startShuttingDown() // Soft shutdown; start draining connections.
	select {
	case err := <-retCh:
		// It shut down fine with just the soft shutdown; everything was
		// well-behaved.  It isn't necessary to cut shutdown short by
		// triggering a hard shutdown with timeToDie() in this case.
		return err
	case <-time.After(5 * time.Second): // shutdown grace period
		// It's taking too long to shut down--it seems that some clients
		// are refusing to hang up.  So now we trigger a hard shutdown
		// and forcefully close the connections.  This will cause errors
		// for those clients.
		timeToDie() // Hard shutdown; cause errors for clients
		return <-retCh
	}
}

// ListenAndServeHTTPWithContext runs server.ListenAndServe() on an http.Server,
// but properly calls server.Shutdown when the Context is canceled.
//
// It obeys hard/soft cancellation as implemented by dcontext.WithSoftness; it
// calls server.Shutdown() when the soft Context is canceled, and the hard
// Context being canceled causes the .Shutdown() to hurry along and kill any
// live requests and return, instead of waiting for them to be completed
// gracefully.
func ListenAndServeHTTPWithContext(ctx context.Context, server *http.Server) error {
	// An HTTP server is a bit of a complex example; for two reasons:
	//
	//  1. Like all network servers, it is a thing that manages multiple
	//     worker goroutines.  Because of this, it is an exception to a
	//     usual rule of Contexts:
	//
	//      > Do not store Contexts inside a struct type; instead, pass a
	//      > Context explicitly to each function that needs it.
	//      >
	//      > -- the "context" package documentation
	//
	//  2. http.Server has its own clunky soft/hard shutdown mechanism, and
	//     a large part of what this function is doing is adapting that to
	//     the less-clunky dcontext mechanism.
	//
	// For those reasons, this isn't necessarily a good instructive example
	// of how to use dcontext, but it is a *real* example.

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
