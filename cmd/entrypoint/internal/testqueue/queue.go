package testqueue

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/datawire/dlib/dgroup"
	"github.com/datawire/dlib/dlog"
)

// The Queue struct implements a multi-writer/multi-reader concurrent queue where the dequeue
// operation (the Get() method) takes a predicate that allows it to skip past queue entries until it
// finds one that satisfies the specified predicate.
type Queue struct {
	timeout time.Duration
	cond    *sync.Cond
	entries []interface{}
	offset  int
}

// NewQueue constructs a new queue with the supplied timeout.
func NewQueue(t *testing.T, timeout time.Duration) *Queue {
	q := &Queue{
		timeout: timeout,
		cond:    sync.NewCond(&sync.Mutex{}),
	}
	ctx, cancel := context.WithCancel(dlog.NewTestContext(t, true))
	grp := dgroup.NewGroup(ctx, dgroup.GroupConfig{})
	t.Cleanup(func() {
		cancel()
		assert.NoError(t, grp.Wait())
	})
	// Broadcast on Queue.cond every three seconds so that anyone waiting on the condition has a
	// chance to timeout. (Go doesn't support timed wait on conditions.)
	grp.Go("ticker", func(ctx context.Context) error {
		ticker := time.NewTicker(3 * time.Second)
		for {
			select {
			case <-ticker.C:
				q.cond.Broadcast()
			case <-ctx.Done():
				return nil
			}
		}
	})
	return q
}

// Add an entry to the queue.
func (q *Queue) Add(t *testing.T, obj interface{}) {
	t.Helper()
	q.cond.L.Lock()
	defer q.cond.L.Unlock()
	q.entries = append(q.entries, obj)
	q.cond.Broadcast()
}

// Get will return the next entry that satisfies the supplied predicate.
func (q *Queue) Get(t *testing.T, predicate func(interface{}) bool) (interface{}, error) {
	t.Helper()
	start := time.Now()
	q.cond.L.Lock()
	defer q.cond.L.Unlock()

	for {
		for idx, obj := range q.entries[q.offset:] {
			if predicate(obj) {
				q.offset += idx + 1
				return obj, nil
			}
		}

		if time.Since(start) > q.timeout {
			msg := &strings.Builder{}
			for idx, entry := range q.entries {
				bytes, err := json.MarshalIndent(entry, "", "  ")
				if err != nil {
					return nil, err
				}
				var extra string
				if idx < q.offset {
					extra = "(Before Offset)"
				} else if idx == q.offset {
					extra = "(Offset Here)"
				} else {
					extra = "(After Offset)"
				}
				msg.WriteString(fmt.Sprintf("\n--- Queue Entry[%d] %s---\n%s\n", idx, extra, string(bytes)))
			}

			t.Fatalf("Get timed out!\n%s", msg)
		}
		q.cond.Wait()
	}
}

// AssertEmpty will check that the queue remains empty for the supplied duration.
func (q *Queue) AssertEmpty(t *testing.T, timeout time.Duration, msg string) {
	t.Helper()
	time.Sleep(timeout)
	q.cond.L.Lock()
	defer q.cond.L.Unlock()
	assert.Empty(t, q.entries, msg)
}
