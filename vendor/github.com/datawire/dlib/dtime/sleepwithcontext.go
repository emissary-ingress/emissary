package dtime

import (
	"context"
	"time"
)

// sleepTestHook is a hook that SleepWithContext calls, that lets us
// insert a pause in order to more reliably test a certain race
// condition.
var sleepTestHook func()

// SleepWithContext pauses the current goroutine for at least the duration d, or
// until the Context is done, whichever happens first.
//
// You may be thinking, why not just do:
//
//     select {
//     case <-ctx.Done():
//     case <-time.After(d):
//     }
//
// well, time.After can't get garbage collected until the timer
// expires, even if the Context is done.  What this function provides
// is properly stopping the timer so that it can be garbage collected
// sooner.
//
// https://medium.com/@oboturov/golang-time-after-is-not-garbage-collected-4cbc94740082
func SleepWithContext(ctx context.Context, d time.Duration) {
	if d <= 0 {
		return
	}
	timer := time.NewTimer(d)
	select {
	case <-ctx.Done():
		if sleepTestHook != nil {
			sleepTestHook()
		}
		if !timer.Stop() {
			<-timer.C
		}
	case <-timer.C:
	}
}
