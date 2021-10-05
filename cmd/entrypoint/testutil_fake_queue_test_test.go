package entrypoint_test

import (
	"testing"
	"time"

	"github.com/datawire/ambassador/v2/cmd/entrypoint"
	"github.com/stretchr/testify/require"
)

func TestFakeQueueGet(t *testing.T) {
	q := entrypoint.NewQueue(t, 10*time.Second)

	go func() {
		for count := 0; count < 10; count++ {
			q.Add(count)
		}
	}()

	for count := 0; count < 10; count++ {
		obj := q.Get(func(obj interface{}) bool {
			return true
		})
		require.Equal(t, count, obj)
	}
}

func TestFakeQueueSkip(t *testing.T) {
	q := entrypoint.NewQueue(t, 10*time.Second)

	go func() {
		for count := 0; count < 10; count++ {
			q.Add(count)
		}
	}()

	for count := 0; count < 10; count += 2 {
		obj := q.Get(func(obj interface{}) bool {
			i := obj.(int)
			return (i % 2) == 0
		})
		require.Equal(t, count, obj)
	}
}
