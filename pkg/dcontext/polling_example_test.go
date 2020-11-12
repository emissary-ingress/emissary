package dcontext_test

import (
	"context"

	"github.com/datawire/ambassador/pkg/dcontext"
)

// This example shows a simple 'examplePollingCallee' that is a worker function that
// takes a Context and uses it to support graceful shutdown.
//
// Unlike the plain "Callee" example, instead of using the <-ctx.Done() channel
// to select when to shut down, it polls ctx.Err() in a loop to decide when to
// shut down.
func Example_pollingCallee() {
	// Ignore this function, it's just here because godoc won't let let you
	// define an example function with arguments.
}

// This is the real example function that you should be paying attention to.
func examplePollingCallee(ctx context.Context) {
	// We assume that ctx is a soft Context

	// Run the main "normal-operation" part of the code until ctx is done.
	// We use the passed-in soft Context as the context for normal
	// operation.
	for ctx.Err() == nil { // ctx.Err() returns nil iff ctx is not done
		DoWork(ctx)
	}

	// Once the soft ctx is done, we use the hard Context as the context for
	// shutdown logic.
	ctx = dcontext.HardContext(ctx)
	DoShutdown(ctx)
}
