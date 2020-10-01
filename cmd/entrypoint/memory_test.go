package entrypoint

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// Test that we trigger "actions" (which we just use to log when interesting stuff happens) at the
// right times.
func TestMemoryUsage(t *testing.T) {
	start := time.Now()
	usage := &MemoryUsage{}

	// nothing interesting has happened yet
	usage.maybeDo(start, func() {
		assert.Fail(t, "no action")
	})

	// memory jumped, but not enough to qualify as interesting
	usage.Usage = 1024 * 1024
	usage.maybeDo(start, func() {
		assert.Fail(t, "no action")
	})

	// memory jumped enough to qualify as interesting
	usage.Usage = 1024 * 1024 * 1024
	did := false
	usage.maybeDo(start, func() {
		did = true
	})
	assert.True(t, did)

	// nothing else interesting has happened
	usage.maybeDo(start, func() {
		assert.Fail(t, "no action")
	})

	// one minute passed, but we are don't have a limit, so we don't care
	usage.maybeDo(start.Add(60*time.Second), func() {
		assert.Fail(t, "no action")
	})

	// we are now over 50% capacity and one minute has passed, so this qualifies as interesting
	usage.Limit = usage.Usage + 1
	did = false
	usage.maybeDo(start.Add(60*time.Second), func() {
		did = true
	})
	assert.True(t, did)

	// we are still over 50% capacity, but we just triggered an action, so this doesn't count as interesting
	usage.maybeDo(start.Add(61*time.Second), func() {
		assert.Fail(t, "no action")
	})

	// but in another 59 seconds, it counts as interesting
	did = false
	usage.maybeDo(start.Add(120*time.Second), func() {
		did = true
	})
	assert.True(t, did)
}
