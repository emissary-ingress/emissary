package dcontext_test

import (
	"context"

	"github.com/datawire/ambassador/pkg/dcontext"
)

// This example shows a simple 'exampleCallee' that is a worker function that
// takes a Context and uses it to support graceful shutdown.
func Example_callee() {
	// Ignore this function, it's just here because godoc won't let let you
	// define an example function with arguments.
}

// This is the real example function that you should be paying attention to.
func exampleCallee(ctx context.Context, datasource <-chan Data) (err error) {
	// We assume that ctx is a soft Context

	defer func() {
		// We use the hard Context as the Context for shutdown logic.
		ctx := dcontext.HardContext(ctx)
		_err := DoShutdown(ctx)
		// Don't hide an error returned by the main part of the work.
		if err == nil {
			err = _err
		}
	}()

	// Run the main "normal-operation" part of the code until ctx is done.
	// We use the passed-in soft Context as the context for normal
	// operation.
	for {
		select {
		case dat := <-datasource:
			if err := DoWorkOnData(ctx, dat); err != nil {
				return err
			}
		case <-ctx.Done():
			return
		}
	}
}
