package entrypoint

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/datawire/dlib/dlog"
)

// The Queue struct implements a multi-writer/multi-reader concurrent queue where the dequeue
// operation (the Get() method) takes a predicate that allows it to skip past queue entries until it
// finds one that satisfies the specified predicate.
type Queue struct {
	T       *testing.T
	timeout time.Duration
	cond    *sync.Cond
	entries []interface{}
	offset  int
}

// NewQueue constructs a new queue with the supplied timeout.
func NewQueue(t *testing.T, timeout time.Duration) *Queue {
	q := &Queue{
		T:       t,
		timeout: timeout,
		cond:    sync.NewCond(&sync.Mutex{}),
	}
	ctx := dlog.NewTestContext(t, false)
	// Broadcast on Queue.cond every three seconds so that anyone waiting on the condition has a
	// chance to timeout. (Go doesn't support timed wait on conditions.)
	go func() {
		ticker := time.NewTicker(3 * time.Second)
		for {
			select {
			case <-ticker.C:
				q.cond.Broadcast()
			case <-ctx.Done():
				return
			}
		}
	}()
	return q
}

// Add an entry to the queue.
func (q *Queue) Add(obj interface{}) {
	q.cond.L.Lock()
	defer q.cond.L.Unlock()
	q.entries = append(q.entries, obj)
	q.cond.Broadcast()
}

// Get will return the next entry that satisfies the supplied predicate.
func (q *Queue) Get(predicate func(interface{}) bool) interface{} {
	q.T.Helper()
	start := time.Now()
	q.cond.L.Lock()
	defer q.cond.L.Unlock()

	for {
		for idx, obj := range q.entries[q.offset:] {
			if predicate(obj) {
				q.offset += idx + 1
				return obj
			}
		}

		if time.Since(start) > q.timeout {
			msg := &strings.Builder{}
			for idx, entry := range q.entries {
				bytes, err := json.MarshalIndent(entry, "", "  ")
				if err != nil {
					panic(err)
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

			q.T.Fatal(fmt.Sprintf("Get timed out!\n%s", msg))
		}
		q.cond.Wait()
	}
}

// AssertEmpty will check that the queue remains empty for the supplied duration.
func (q *Queue) AssertEmpty(timeout time.Duration, msg string) {
	q.T.Helper()
	time.Sleep(timeout)
	q.cond.L.Lock()
	defer q.cond.L.Unlock()
	assert.Empty(q.T, q.entries, msg)
}
