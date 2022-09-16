package memory

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/datawire/dlib/dlog"
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
	usage.usage = 1024 * 1024
	usage.maybeDo(start, func() {
		assert.Fail(t, "no action")
	})

	// memory jumped enough to qualify as interesting
	usage.usage = 1024 * 1024 * 1024
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
	usage.limit = usage.usage + 1
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
	ctx := dlog.NewTestContext(t, false)
	count := 0
	m := &MemoryUsage{
		limit:      unlimited,
		perProcess: map[int]*ProcessUsage{},
		readUsage: func(_ context.Context) (memory, memory) {
			return 0, unlimited
		},
		readPerProcess: func(_ context.Context) map[int]*ProcessUsage {
			defer func() {
				count = count + 1
			}()
			switch count {
			case 0:
				return map[int]*ProcessUsage{
					1: {1, []string{"one"}, 1024, 0},
					2: {2, []string{"two"}, 1024, 0},
					3: {3, []string{"three"}, 1024, 0},
					4: {4, []string{"four"}, 1024, 0},
					5: {5, []string{"five"}, 1024, 0},
				}
			case 1:
				return map[int]*ProcessUsage{
					1: {1, []string{"one"}, 1024, 0},
					2: {2, []string{"two"}, 1024, 0},
					4: {4, []string{"four"}, 1024, 0},
					5: {5, []string{"five"}, 1024, 0},
				}
			case 2:
				return map[int]*ProcessUsage{
					1: {1, []string{"one"}, 1024, 0},
					2: {2, []string{"two"}, 1024, 0},
					5: {5, []string{"five"}, 1024, 0},
				}
			case 3:
				return map[int]*ProcessUsage{
					1: {1, []string{"one"}, 1024, 0},
					5: {5, []string{"five"}, 1024, 0},
				}
			default:
				return map[int]*ProcessUsage{
					1: {1, []string{"one"}, 1024, 0},
					5: {5, []string{"five"}, 1024, 0},
				}
			}
		},
	}

	t.Log(m.String())

	assert.Equal(t, 0, len(m.perProcess))
	m.Refresh(ctx)
	t.Log(m.String())
	assert.Equal(t, 5, len(m.perProcess))
	m.Refresh(ctx)
	t.Log(m.String())
	for i := 0; i < 10; i++ {
		assert.Equal(t, 5, len(m.perProcess))
		m.Refresh(ctx)
		t.Log(m.String())
	}

	m.Refresh(ctx)
	assert.Equal(t, 4, len(m.perProcess))
	t.Log(m.String())
	assert.NotContains(t, m.perProcess, 3)

	m.Refresh(ctx)
	assert.Equal(t, 3, len(m.perProcess))
	t.Log(m.String())
	assert.NotContains(t, m.perProcess, 4)

	m.Refresh(ctx)
	assert.Equal(t, 2, len(m.perProcess))
	t.Log(m.String())
	assert.NotContains(t, m.perProcess, 2)

	m.Refresh(ctx)
	assert.Equal(t, 2, len(m.perProcess))
	t.Log(m.String())
	assert.Contains(t, m.perProcess, 1)
	assert.Contains(t, m.perProcess, 5)

}

func TestParseMemoryStat(t *testing.T) {
	assert := assert.New(t)
	contents := `
cache 175247360
rss 403296256
rss_huge 65011712
shmem 0
mapped_file 93401088
dirty 0
writeback 0
swap 1
pgpgin 5829351
pgpgout 5726886
pgfault 5848359
pgmajfault 792
inactive_anon 0
active_anon 309968896
inactive_file 222568448
active_file 46092288
unevictable 0
hierarchical_memory_limit 2097152000
hierarchical_memsw_limit 9223372036854771712
total_cache 175247360
total_rss 403296256
total_rss_huge 65011712
total_shmem 0
total_mapped_file 93401088
total_dirty 0
total_writeback 0
total_swap 0
total_pgpgin 5829351
total_pgpgout 5726886
total_pgfault 5848359
total_pgmajfault 792
total_inactive_anon 0
total_active_anon 309968896
total_inactive_file 222568448
total_active_file 46092288
total_unevictable 0
`
	result, err := parseMemoryStat(contents)
	assert.NoError(err)
	assert.Equal(uint64(403296256), result.Rss)
	assert.Equal(uint64(175247360), result.Cache)
	assert.Equal(uint64(1), result.Swap)
	assert.Equal(uint64(222568448), result.InactiveFile)
}
