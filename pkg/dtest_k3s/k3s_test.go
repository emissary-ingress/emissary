package dtest_k3s

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/datawire/dlib/dlog"
)

func TestMain(m *testing.M) {
	// we get the lock to make sure we are the only thing running
	// because the nat tests interfere with docker functionality
	WithMachineLock(context.TODO(), func(ctx context.Context) {
		os.Exit(m.Run())
	})
}

func TestContainer(t *testing.T) {
	ctx := dlog.NewTestContext(t, false)
	id := dockerUp(ctx, "dtest-test-tag", "nginx")

	running := dockerPs(ctx)
	assert.Contains(t, running, id)

	dockerKill(ctx, id)

	running = dockerPs(ctx)
	assert.NotContains(t, running, id)
}
