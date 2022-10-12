package debug

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"
)

// The Timer struct can be used to time discrete actions. It tracks min, max, average, and total
// elapsed time for all actions. The Timer struct is thread safe, and the Start() method is designed
// to do proper bookkeeping regardless of concurrent use from multiple goroutines.
//
// Example 1
//
//	var GlobalTimer = NewTimer()
//
//	func foo() {
//	  GlobalTimer.Time(func() {
//	    // ...
//	  })
//	}
//
// Example 2
//
//	func foo() {
//	  stop := GlobalTimer.Start()
//	  defer stop()
//	  ...
//	}
//
// Example 3
//
//	func foo() {
//	  defer GlobalTimer.Start()()
//	  ...
//	}
type Timer struct {
	mutex sync.Mutex    // protects the whole struct
	count int           // counts the number of actions that have been timed
	total time.Duration // tracks the total elapsed time for all actions
	min   time.Duration // the max elapsed time for an action
	max   time.Duration // the min elapsed time for an action

	clock func() time.Time // The clock function used by the timer.
}

// The type of the clock function to use for timing.
type ClockFunc func() time.Time

// The type of the function used to stop timing.
type StopFunc func()

// The NewTimer function creates a new timer. It uses time.Now as the clock for the timer.
func NewTimer() *Timer {
	return NewTimerWithClock(time.Now)
}

// The NewTimerWithClock function creates a new timer with the given name and clock.
func NewTimerWithClock(clock ClockFunc) *Timer {
	return &Timer{
		clock: clock,
	}
}

// The Start() method starts timing an action.
func (t *Timer) Start() StopFunc {
	start := t.clock()
	return func() {
		t.record(start, t.clock())
	}
}

// Records the timing info for an action.
func (t *Timer) record(start time.Time, stop time.Time) {
	t.withMutex(func() {
		delta := stop.Sub(start)

		if t.count == 0 {
			// Initialize min and max if this is the first event.
			t.min = delta
			t.max = delta
		} else {
			// Update min and max for subsequent events.
			if delta < t.min {
				t.min = delta
			}
			if delta > t.max {
				t.max = delta
			}
		}

		t.count++
		t.total += delta
	})
}

// Convenience function for safely accessing the internals of the struct.
func (t *Timer) withMutex(f func()) {
	t.mutex.Lock()
	defer t.mutex.Unlock()
	f()
}

// The Copy() method returns a copy of the timer. This can be used to get a consistent snapshot of
// all the Count/Min/Max/Average/Total values.
func (t *Timer) Copy() (result *Timer) {
	t.withMutex(func() {
		result = &Timer{}
		*result = *t // nolint:govet // silence complaint about copying t.mutex
		result.mutex = sync.Mutex{}
	})
	return
}

// The Count() method returns the number of events that have been timed.
func (t *Timer) Count() (result int) {
	t.withMutex(func() {
		result = t.count
	})
	return
}

// The Min() method retuns the minimum duration of all timed events.
func (t *Timer) Min() (result time.Duration) {
	t.withMutex(func() {
		result = t.min
	})
	return
}

// The Max() method returns the maximum duration of all timed events.
func (t *Timer) Max() (result time.Duration) {
	t.withMutex(func() {
		result = t.max
	})
	return
}

// The Average() method returns the average duration of all timed events.
func (t *Timer) Average() (result time.Duration) {
	t.withMutex(func() {
		if t.count > 0 {
			result = t.total / time.Duration(t.count)
		} else {
			result = 0
		}
	})
	return
}

// The Total() method returns the total duration of all events.
func (t *Timer) Total() (result time.Duration) {
	t.withMutex(func() {
		result = t.total
	})
	return
}

// The Time() method is a convenience method that times invocation of the supplied function.
func (t *Timer) Time(f func()) {
	defer t.Start()()
	f()
}

// The TimedHandler() method wraps the supplied Handler with a Handler that times every request.
func (t *Timer) TimedHandler(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer t.Start()()
		h.ServeHTTP(w, r)
	})
}

// The TimedHandlerFunc() method wraps the supplied HandlerFunc with a HandlerFunc that times every request.
func (t *Timer) TimedHandlerFunc(h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		defer t.Start()()
		h(w, r)
	}
}

func (t *Timer) MarshalJSON() ([]byte, error) {
	c := t.Copy()
	return json.Marshal(fmt.Sprintf("%d, %s/%s/%s", c.Count(), c.Min().String(), c.Average().String(), c.Max().String()))
}
