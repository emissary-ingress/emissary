package ambex

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/datawire/dlib/dlog"
)

type harness struct {
	T               *testing.T
	C               chan time.Time
	version         int         // for generating versions
	updates         chan Update // we push versions here
	pushed          chan int    // versions that are pushed end up here
	expectedVersion int

	mutex sync.Mutex // to proect the clock and usage
	usage int        // simulated memory usage
	clock time.Time  // current simulated time
}

var drainTime = 10 * time.Minute

func newHarness(t *testing.T) *harness {
	C := make(chan time.Time)
	h := &harness{t, C, 0, make(chan Update), make(chan int, 10000), 1, sync.Mutex{}, 0, time.Now()}
	go func() {
		assert.NoError(t, updaterWithTicker(dlog.NewTestContext(t, false), h.updates, h.getUsage, drainTime, &time.Ticker{C: C}, h.time))
	}()
	return h
}

func (h *harness) advance(d time.Duration) time.Time {
	h.mutex.Lock()
	result := h.clock.Add(d)
	h.clock = result
	h.mutex.Unlock()
	return result
}

func (h *harness) time() time.Time {
	return h.advance(0)
}

func (h *harness) getUsage() int {
	h.mutex.Lock()
	defer h.mutex.Unlock()
	return h.usage
}

func (h *harness) setUsage(u int) {
	h.mutex.Lock()
	defer h.mutex.Unlock()
	h.usage = u
}

// Simulate a timer tick after the given duration.
func (h *harness) tick(d time.Duration) {
	h.C <- h.advance(d)
}

// Simulate an update with the specified version after the specified duration.
func (h *harness) update(d time.Duration) int {
	h.version++
	version := h.version
	h.advance(d)
	h.updates <- Update{fmt.Sprintf("%d", version), func() error {
		h.pushed <- version
		return nil
	}}
	return version
}

// Assert that a contiguous sequences of updates were pushed up to and including the specified
// version.
func (h *harness) expectUntil(version int) {
	for {
		select {
		case v := <-h.pushed:
			assert.Equal(h.T, h.expectedVersion, v)
			if v < version {
				h.expectedVersion++
				continue
			}

			if v != version {
				assert.Fail(h.T, fmt.Sprintf("expected version %d, got %d", version, v))
			}
			return
		case <-time.After(1 * time.Second):
			assert.Fail(h.T, fmt.Sprintf("expected version %d, but timed out", h.expectedVersion))
			return
		}
	}

}

// Assert that the next version is exactly the supplied version.
func (h *harness) expectExact(version int) {
	select {
	case v := <-h.pushed:
		assert.Equal(h.T, version, v)
		h.expectedVersion = version + 1
	case <-time.After(1 * time.Second):
		assert.Fail(h.T, fmt.Sprintf("expected version %d, but timed out", version))
		return
	}
}

// Assert that no version was pushed.
func (h *harness) expectNone() {
	select {
	case v := <-h.pushed:
		assert.Fail(h.T, fmt.Sprintf("expected no pushes, but go version %d", v))
	case <-time.After(1 * time.Second):
		return
	}
}

// Check to see when memory usage is zero we don't throttle at all.
func TestHappyPath(t *testing.T) {
	h := newHarness(t)
	var version int
	for i := 0; i < 1000; i++ {
		version = h.update(0)
	}
	h.expectUntil(version)
}

// Progress through the various levels of constraint and check that the correct updates are dropped.
func TestConstrained(t *testing.T) {
	h := newHarness(t)

	h.setUsage(50)
	for i := 0; i < 1000; i++ {
		h.update(0)
	}
	// Above 50% memory usage we only allow for 120 stale configs in memory at a time. Our harness
	// versions start counting at 1, so we should get up to version 120 before getting throttled.
	h.expectUntil(120)
	// Fast forward by drainTime and check that the most recent update made it eventually.
	h.tick(drainTime)
	h.expectExact(1000)

	h.setUsage(60)
	for i := 0; i < 1000; i++ {
		h.update(0)
	}
	// Above 60% memory usage we only allow for 60 stale configs in memory at a time.
	h.expectUntil(1059)
	// Fast forward by drainTime and check that the most recent update made it eventually.
	h.tick(drainTime)
	h.expectExact(2000)

	h.setUsage(70)
	for i := 0; i < 1000; i++ {
		h.update(0)
	}
	// Above 70% memory usage we only allow for 30 stale configs in memory at a time.
	h.expectUntil(2029)
	// Fast forward by drainTime and check that the most recent update made it eventually.
	h.tick(drainTime)
	h.expectExact(3000)

	h.setUsage(80)
	for i := 0; i < 1000; i++ {
		h.update(0)
	}
	// Above 80% memory usage we only allow for 15 stale configs in memory at a time.
	h.expectUntil(3014)
	// Fast forward by drainTime and check that the most recent update made it eventually.
	h.tick(drainTime)
	h.expectExact(4000)

	h.setUsage(90)
	for i := 0; i < 1000; i++ {
		h.update(0)
	}
	// Above 90% memory usage we only allow for 1 stale config in memory at a time.
	h.expectNone()
	// Fast forward by drainTime and check that the most recent update made it eventually.
	h.tick(drainTime)
	h.expectExact(5000)

	// Check that we go back to passing through everything when usage drops again.
	h.setUsage(25)
	for i := 0; i < 1000; i++ {
		h.update(0)
	}
	h.expectUntil(6000)
}
