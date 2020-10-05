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

// Make sure we clear out exited processes after 10 refreshes.
func TestMemoryUsageGCExited(t *testing.T) {
	count := 0
	m := &MemoryUsage{
		PerProcess: map[int]*ProcessUsage{},
		readUsage: func() (memory, memory) {
			return 0, 0
		},
		perProcess: func() map[int]*ProcessUsage {
			defer func() {
				count = count + 1
			}()
			switch count {
			case 0:
				return map[int]*ProcessUsage{
					1: &ProcessUsage{1, []string{"one"}, 1024, 0},
					2: &ProcessUsage{2, []string{"two"}, 1024, 0},
					3: &ProcessUsage{3, []string{"three"}, 1024, 0},
					4: &ProcessUsage{4, []string{"four"}, 1024, 0},
					5: &ProcessUsage{5, []string{"five"}, 1024, 0},
				}
			case 1:
				return map[int]*ProcessUsage{
					1: &ProcessUsage{1, []string{"one"}, 1024, 0},
					2: &ProcessUsage{2, []string{"two"}, 1024, 0},
					4: &ProcessUsage{4, []string{"four"}, 1024, 0},
					5: &ProcessUsage{5, []string{"five"}, 1024, 0},
				}
			case 2:
				return map[int]*ProcessUsage{
					1: &ProcessUsage{1, []string{"one"}, 1024, 0},
					2: &ProcessUsage{2, []string{"two"}, 1024, 0},
					5: &ProcessUsage{5, []string{"five"}, 1024, 0},
				}
			case 3:
				return map[int]*ProcessUsage{
					1: &ProcessUsage{1, []string{"one"}, 1024, 0},
					5: &ProcessUsage{5, []string{"five"}, 1024, 0},
				}
			default:
				return map[int]*ProcessUsage{
					1: &ProcessUsage{1, []string{"one"}, 1024, 0},
					5: &ProcessUsage{5, []string{"five"}, 1024, 0},
				}
			}
		},
	}

	t.Log(m.String())

	assert.Equal(t, 0, len(m.PerProcess))
	m.Refresh()
	t.Log(m.String())
	assert.Equal(t, 5, len(m.PerProcess))
	m.Refresh()
	t.Log(m.String())
	for i := 0; i < 10; i++ {
		assert.Equal(t, 5, len(m.PerProcess))
		m.Refresh()
		t.Log(m.String())
	}

	m.Refresh()
	assert.Equal(t, 4, len(m.PerProcess))
	t.Log(m.String())
	assert.NotContains(t, m.PerProcess, 3)

	m.Refresh()
	assert.Equal(t, 3, len(m.PerProcess))
	t.Log(m.String())
	assert.NotContains(t, m.PerProcess, 4)

	m.Refresh()
	assert.Equal(t, 2, len(m.PerProcess))
	t.Log(m.String())
	assert.NotContains(t, m.PerProcess, 2)

	m.Refresh()
	assert.Equal(t, 2, len(m.PerProcess))
	t.Log(m.String())
	assert.Contains(t, m.PerProcess, 1)
	assert.Contains(t, m.PerProcess, 5)

}
