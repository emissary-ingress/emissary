package dcontext_test

import (
	"context"

	"github.com/datawire/ambassador/pkg/dcontext"
)

func Example_callee(ctx context.Context, datasource <-chan Data) (err error) {
	// We assume that ctx is a soft Context

	defer func() {
		// We use the hard Context as the Context for shutdown
		// logic.
		ctx := dcontext.HardContext(ctx)
		_err := DoShutdown(ctx)
		// Don't hide an error returned by the main part of
		// the work.
		if err == nil {
			err = _err
		}
	}()

	// Run the main normal-operation part of the code until ctx is
	// done.  We use the passed-in soft Context as the context for
	// normal operation.
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

func Example_pollingCallee(ctx context.Context) {
	// We assume that ctx is a soft Context

	// Run the main normal-operation part of the code until ctx is
	// done.  We use the passed-in soft Context as the context for
	// normal operation.
	for ctx.Err() == nil { // ctx.Err() returns nil iff ctx is not done
		DoWork(ctx)
	}

	// Once the soft ctx is done, we use the hard Context as the
	// context for shutdown logic.
	ctx = dcontext.HardContext(ctx)
	DoShutdown(ctx)
}

type Data struct{}

func DoWork(_ context.Context) error {}

func DoWorkOnData(_ context.Context, _ Data) error {}

func DoShutdown(_ context.Context) error {}
