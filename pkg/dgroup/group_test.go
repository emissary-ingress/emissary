package dgroup

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

// SetStacktraceForTesting overrides the stacktrace that would be
// logged by dgroup, to set it to something fixed, to make dgroup's
// unit tests simpler.
func SetStacktraceForTesting(trace string) {
	stacktraceForTesting = trace
}

func TestParentGroup(t *testing.T) {
	// The example tests the positive case, so just test the
	// negative case here.
	group := ParentGroup(context.Background())
	assert.Nil(t, group)
}
