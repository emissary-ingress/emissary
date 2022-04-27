package debug_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/emissary-ingress/emissary/v3/pkg/debug"
)

func TestTimer(t *testing.T) {
	clock := time.Now()
	timer := debug.NewTimerWithClock(func() time.Time {
		return clock
	})

	timer.Time(func() {
		clock = clock.Add(250 * time.Millisecond)
	})
	timer.Time(func() {
		clock = clock.Add(250 * time.Millisecond)
	})
	timer.Time(func() {
		clock = clock.Add(500 * time.Millisecond)
	})
	timer.Time(func() {
		clock = clock.Add(500 * time.Millisecond)
	})
	timer.Time(func() {
		clock = clock.Add(500 * time.Millisecond)
	})

	assert.Equal(t, 5, timer.Count())
	assert.Equal(t, 250*time.Millisecond, timer.Min())
	assert.Equal(t, 500*time.Millisecond, timer.Max())
	assert.Equal(t, 400*time.Millisecond, timer.Average())
}

func TestConcurrentTiming(t *testing.T) {
	clock := time.Now()
	timer := debug.NewTimerWithClock(func() time.Time {
		return clock
	})

	// Two simultaneous starts
	stop1 := timer.Start()
	stop2 := timer.Start()

	// First one stops after 250 milliseconds.
	clock = clock.Add(250 * time.Millisecond)
	stop1()

	// Second one stops after 750 milliseconds (the prior 250 + an additional 500).
	clock = clock.Add(500 * time.Millisecond)
	stop2()

	assert.Equal(t, 2, timer.Count())
	assert.Equal(t, 250*time.Millisecond, timer.Min())
	assert.Equal(t, 750*time.Millisecond, timer.Max())
	assert.Equal(t, 500*time.Millisecond, timer.Average())
}

func TestAverageZero(t *testing.T) {
	assert.Equal(t, 0*time.Second, debug.NewTimer().Average())
}
