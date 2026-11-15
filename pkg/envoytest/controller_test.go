package envoytest

import (
	"context"
	"sync"
	"testing"

	ecp_v3_cache "github.com/envoyproxy/go-control-plane/pkg/cache/v3"

	"github.com/datawire/dlib/dlog"
)

// NewEnvoyController hands ecLogger to the snapshot cache before anything calls
// Run, so the cache can log from whichever goroutine touches it while Run is
// still capturing the context those log calls read. Drive both at once: under
// -race an unsynchronized handoff fails here, and this reaches it through the
// same path as pkg/gateway's TestGatewayMatches, which skips on darwin and so
// cannot be the guard on a Mac.
func TestEnvoyControllerLogCtxRace(t *testing.T) {
	ctx, cancel := context.WithCancel(dlog.NewTestContext(t, false))
	defer cancel()

	// Port 0 so the listener never collides with anything already running.
	ec := NewEnvoyController("127.0.0.1:0")

	start := make(chan struct{})
	snapshotsDone := make(chan struct{})
	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		<-start
		// Serves until ctx is cancelled below; that shutdown is not a failure.
		_ = ec.Run(ctx)
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		defer close(snapshotsDone)
		<-start
		// Every SetSnapshot logs through ecLogger, which is the read side.
		for i := 0; i < 500; i++ {
			if err := ec.configCache.SetSnapshot(ctx, "race-node", &ecp_v3_cache.Snapshot{}); err != nil {
				return
			}
		}
	}()

	close(start)

	// Cancel only once the read side is done, otherwise Run serves forever and
	// wg.Wait never returns.
	<-snapshotsDone
	cancel()
	wg.Wait()
}

// The callbacks can log before Run has captured anything, so a reader must cope
// with no context rather than dereference a nil one.
func TestEnvoyControllerLogsBeforeRun(t *testing.T) {
	ec := NewEnvoyController("127.0.0.1:0")
	logger := ecLogger{ec: ec}

	if got := logger.logCtx(); got == nil {
		t.Fatal("logCtx() returned nil before Run; callbacks would panic")
	}

	logger.Debugf("debug %d", 1)
	logger.Infof("info %d", 2)
	logger.Warnf("warn %d", 3)
	logger.Errorf("error %d", 4)
}
