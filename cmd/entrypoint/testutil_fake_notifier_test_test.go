package entrypoint_test

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/emissary-ingress/emissary/v3/cmd/entrypoint"
)

func TestFakeNotifier(t *testing.T) {
	n := entrypoint.NewNotifier()

	// These will track invocations of our two test listeners. One test listener will be created
	// early, prior to any changes, the other will be created later after changes have occurred.
	earlyCh := make(chan string)
	lateCh := make(chan string)

	stopEarly := n.Listen(generator("early", earlyCh))

	// No listener should be notified until we signal it.
	require.Equal(t, "", get(earlyCh))
	require.Equal(t, "", get(lateCh))
	n.Changed()
	require.Equal(t, "", get(earlyCh))
	require.Equal(t, "", get(lateCh))

	// Send notifications and check that we saw one for the early listener.
	n.Notify()
	require.Equal(t, "early-1", get(earlyCh))
	require.Equal(t, "", get(lateCh))

	// Create the late listener and verify that it "catches up".
	stopLate := n.Listen(generator("late", lateCh))
	require.Equal(t, "", get(earlyCh))
	require.Equal(t, "late-1", get(lateCh))

	// Check that multiple changes will get batched.
	n.Changed()
	n.Changed()

	n.Notify()
	require.Equal(t, "early-2", get(earlyCh))
	require.Equal(t, "late-2", get(lateCh))

	// No changes have happened.
	n.Notify()
	require.Equal(t, "", get(earlyCh))
	require.Equal(t, "", get(lateCh))

	// Both should get notified.
	n.Changed()
	n.Notify()
	require.Equal(t, "early-3", get(earlyCh))
	require.Equal(t, "late-3", get(lateCh))

	// Check auto notification.
	n.AutoNotify(true)
	n.Changed()
	require.Equal(t, "early-4", get(earlyCh))
	require.Equal(t, "late-4", get(lateCh))

	// Check going back to manual notify.
	n.AutoNotify(false)
	n.Changed()
	require.Equal(t, "", get(earlyCh))
	require.Equal(t, "", get(lateCh))
	n.Notify()
	require.Equal(t, "early-5", get(earlyCh))
	require.Equal(t, "late-5", get(lateCh))

	// Now let's stop and see that there are no notifications.
	stopEarly()
	stopLate()
	n.Changed()
	n.Notify()
	require.Equal(t, "", get(earlyCh))
	require.Equal(t, "", get(lateCh))
}

func generator(name string, ch chan string) func() {
	count := 0
	return func() {
		count += 1
		go func() {
			ch <- fmt.Sprintf("%s-%d", name, count)
		}()
	}
}

var timeout time.Duration

func init() {
	// Use a longer timeout in CI to avoid flakes.
	if os.Getenv("CI") != "" {
		timeout = 1 * time.Second
	} else {
		timeout = 100 * time.Millisecond
	}
}

func get(ch chan string) string {
	select {
	case result := <-ch:
		return result
	case <-time.After(timeout):
		return ""
	}
}
