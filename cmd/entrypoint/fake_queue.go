package entrypoint

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"testing"
	"time"
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
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
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
			fmt.Println("GET TIMED OUT")

			if q.offset >= len(q.entries) {
				fmt.Println("--- Queue is empty ---")
			} else {
				// Walk the outstanding entries in the queue and dump them as
				// JSON, so that the test writer has a fighting chance of
				// figuring out _why_ the get has timed out.
				for i := q.offset; i < len(q.entries); i++ {
					bytes, err := json.MarshalIndent(q.entries[i], "", "  ")

					if err != nil {
						panic(err)
					}

					fmt.Printf("--- Queue Entry %d ---\n", i)
					fmt.Println(string(bytes))
				}
			}

			q.T.Fatal("Get timed out!")
		}
		q.cond.Wait()
	}
}
