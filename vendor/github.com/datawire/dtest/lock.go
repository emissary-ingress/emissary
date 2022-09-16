package dtest

import (
	"context"
	"fmt"
	"os"
)

func exit(filename string, err error) {
	fmt.Fprintf(os.Stderr, "error trying to acquire lock on %s: %v\n", filename, err)
	os.Exit(1)
}

// WithMachineLock executes the supplied body with a guarantee that it
// is the only code running (via WithMachineLock) on the machine.
func WithMachineLock(ctx context.Context, body func(context.Context)) {
	WithNamedMachineLock(ctx, "default", body)
}
