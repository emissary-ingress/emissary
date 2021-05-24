package dtime

import (
	"time"
)

// FakeTime keeps track of fake time for us, so that we don't have to rely
// on the real system clock. This can make life during testing much, much
// easier -- rather than needing to wait forever, you can control the
// passage of time however you like.
//
// To use FakeTime, use NewFakeTime to instantiate it, then Step (or StepSec)
// to change its current time. FakeTime also remembers its boot time (the
// time when it was instantiated) so that you can meaningfully talk about
// how much fake time has passed since boot and, if necessary, relate fake
// times to actual system times.
type FakeTime struct {
	bootTime    time.Time
	currentTime time.Time
}

// NewFakeTime creates a new FakeTime structure, booted at the current time.
// Once instantiated, its Now method is a drop-in replacement for time.Now.
func NewFakeTime() *FakeTime {
	ft := &FakeTime{}

	ft.bootTime = time.Now()
	ft.currentTime = ft.bootTime

	return ft
}

// Step steps a FakeTime by the given duration. Any duration may be used,
// with all the obvious concerns about stepping the fake clock into the
// past.
func (f *FakeTime) Step(d time.Duration) {
	f.currentTime = f.currentTime.Add(d)
}

// StepSec steps a FakeTime by a given number of seconds. Any number of
// seconds is valid, with all the obvious concerns about stepping the
// fake clock into the past.
//
// This is a convenience to allow writing unit tests that don't have to
// have "* time.Second" scattered over and over and over again through
// everything.
func (f *FakeTime) StepSec(s int) {
	f.Step(time.Duration(s) * time.Second)
}

// BootTime returns the real system time at which the FakeTime was
// instantiated, in case it's needed.
//
// This is an accessor because we don't really want people changing the
// boot time after boot.
func (f *FakeTime) BootTime() time.Time {
	return f.bootTime
}

// Now returns the current fake time. It is a drop-in replacement for
// time.Now, and is particularly suitable for use with dtime.SetNow and
// dtime.Now.
func (f *FakeTime) Now() time.Time {
	return f.currentTime
}

// TimeSinceBoot returns the amount of fake time that has passed since
// the FakeTime was instantiated.
func (f *FakeTime) TimeSinceBoot() time.Duration {
	return f.currentTime.Sub(f.bootTime)
}
