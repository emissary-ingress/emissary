package testqueue_test

import (
	"testing"
	"time"

	"github.com/datawire/ambassador/v2/cmd/entrypoint/internal/testqueue"
	"github.com/stretchr/testify/require"
)

func TestFakeQueueGet(t *testing.T) {
	q := testqueue.NewQueue(t, 10*time.Second)

	go func() {
		for count := 0; count < 10; count++ {
			q.Add(count)
		}
	}()

	for count := 0; count < 10; count++ {
		obj, err := q.Get(func(obj interface{}) bool {
			return true
		})
		require.NoError(t, err)
		require.Equal(t, count, obj)
	}
}

func TestFakeQueueSkip(t *testing.T) {
	q := testqueue.NewQueue(t, 10*time.Second)

	go func() {
		for count := 0; count < 10; count++ {
			q.Add(count)
		}
	}()

	for count := 0; count < 10; count += 2 {
		obj, err := q.Get(func(obj interface{}) bool {
			i := obj.(int)
			return (i % 2) == 0
		})
		require.NoError(t, err)
		require.Equal(t, count, obj)
	}
}
