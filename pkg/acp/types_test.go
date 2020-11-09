package acp_test

import (
	"time"
)

// fakeTime keeps track of fake time for us, so that we don't have to rely
// on the real system clock.
type fakeTime struct {
	bootTime    time.Time
	currentTime time.Time
}

// newFakeTime: create a new fakeTime structure, booted at the current time.
func newFakeTime() *fakeTime {
	ft := &fakeTime{}

	ft.bootTime = time.Now()
	ft.currentTime = ft.bootTime

	return ft
}

// Step a fakeTime by a given number of seconds.
//
// This does _not_ take a time.Duration because it gets called a lot in the
// tests, and I just couldn't take scattering "* time.Second" over and over
// again through the tests.
func (f *fakeTime) stepSec(s int) {
	f.currentTime = f.currentTime.Add(time.Duration(s) * time.Second)
}

// now returns the current fake time.
func (f *fakeTime) now() time.Time {
	return f.currentTime
}

// timeSinceBoot returns the amount of fake time that has passed since boot.
func (f *fakeTime) timeSinceBoot() time.Duration {
	return f.currentTime.Sub(f.bootTime)
}
