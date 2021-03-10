// Copyright 2020 Datawire. All rights reserved.
//
// package acp contains stuff dealing with the Ambassador Control Plane as a whole.
//
// This is the DiagdWatcher, which is a class that can keep an eye on a running
// diagd - and just diagd, all other Ambassador elements are ignored - and tell you
// whether it's alive and ready, or not.
//
// THE GRACE PERIOD:
// Much of DiagdWatcher is concerned with feeding a snapshot to diagd for processing,
// and then noting that processing is done. This can take awhile. Currently, we give
// diagd _ten minutes_ to get its act together, with the ideas that:
//
// a. We really don't want to start summarily killing pods when, say, configuration
//    times go from 30 seconds to 31 seconds, but
// b. We also don't want a dead diagd to hork a pod for hours.
//
// Ten minutes might not be the right compromise, but the idea is that a reasonable
// customer should be doing health checks on their ability to route to their services,
// as well as using the blunt-instrument Kubernetes checks. So this code is more about
// providing a conservative failsafe.
//
// TESTING HOOKS:
// Since time plays a role, you can use DiagdWatcher.SetFetchTime to change the
// function that the DiagdWatcher uses to fetch times. The default is time.Now.
//
// This hook is NOT meant for you to change the values on the fly in a running
// DiagdWatcher. Set it at instantiation if need be, then leave it alone. See
// diagd_test.go for more.

package acp

import (
	"sync"
	"time"
)

// DiagdWatcher encapsulates state and methods for keeping an eye on a running
// diagd, and deciding if it's healthy.
type DiagdWatcher struct {
	// How shall we fetch the current time?
	fetchTime timeFetcher

	// This mutex protects access to LastSent and LastProcessed,
	// mostly as a matter of rank paranoia.
	mutex sync.Mutex

	// When did we last send a snapshot to diagd?
	LastSent time.Time

	// When did we last hear that diagd had processed a snapshot?
	LastProcessed time.Time

	// When does our grace period end? The grace period is ten minutes after
	// the most recent event (boot, or the last time a snapshot was sent).
	GraceEnd time.Time
}

// NewDiagdWatcher creates a new DiagdWatcher.
func NewDiagdWatcher() *DiagdWatcher {
	w := &DiagdWatcher{fetchTime: time.Now}
	w.setGraceEnd(w.fetchTime(), 10*time.Minute) // initial boot grace period

	return w
}

// setGraceEnd will set the end of the grace period to some duration after
// a given timestamp.
func (w *DiagdWatcher) setGraceEnd(start time.Time, dur time.Duration) {
	w.GraceEnd = start.Add(dur)
}

// withinGracePeriod will return true IFF we're within the current grace period.
func (w *DiagdWatcher) withinGracePeriod() bool {
	return w.fetchTime().Before(w.GraceEnd)
}

// SetFetchTime will change the function we use to get the current time _AND RESETS THE
// BOOT GRACE PERIOD_. This is here for testing, _NOT_ to allow switching timers on the
// fly for some crazy reason.
func (w *DiagdWatcher) SetFetchTime(fetchTime timeFetcher) {
	w.fetchTime = fetchTime

	// See comment above for why it's OK to reset the boot grace period here.
	w.setGraceEnd(w.fetchTime(), 10*time.Minute) // RESET boot grace period, see above.
}

// NoteSnapshotSent marks the time at which we have sent a snapshot.
func (w *DiagdWatcher) NoteSnapshotSent() {
	w.mutex.Lock()
	defer w.mutex.Unlock()

	// Remember that we've sent a snapshot...
	w.LastSent = w.fetchTime()

	// ...and reset the grace period IFF we've processed something.
	//
	// Why not do this unconditionally? Basically we don't want an Ambassador
	// to somehow send snapshots over and over, never process any, and not
	// give up. (This situation is currently "impossible", so this is kind of
	// paranoia, but that's OK.)

	if !w.LastProcessed.IsZero() {
		w.setGraceEnd(w.LastSent, 10*time.Minute) // Update grace period
	}
}

// NoteSnapshotProcessed marks the time at which we have processed a snapshot.
func (w *DiagdWatcher) NoteSnapshotProcessed() {
	w.mutex.Lock()
	defer w.mutex.Unlock()
	w.LastProcessed = w.fetchTime()
}

// IsAlive returns true IFF diagd should be considered alive.
func (w *DiagdWatcher) IsAlive() bool {
	w.mutex.Lock()
	defer w.mutex.Unlock()

	// Case 1: we've sent and processed at least one snapshot, and LastSent is before
	// LastProcessed. This is the case where the last-sent snapshot has already been
	// processed -- life is good.
	if !w.LastSent.IsZero() && !w.LastProcessed.IsZero() && w.LastSent.Before(w.LastProcessed) {
		// Yes -- both LastSent and LastProcessed are set, and LastSent is before LastProcessed.
		// We're good to go.
		return true
	}

	// Case 2: the above isn't true. Either we haven't tried to send a snapshot yet, or
	// we've sent a snapshot and haven't finished processing it yet. In either case, we'll
	// say we're alive only as long as we're within the grace period.
	return w.withinGracePeriod()
}

// IsReady returns true IFF diagd should be considered ready.
func (w *DiagdWatcher) IsReady() bool {
	w.mutex.Lock()
	defer w.mutex.Unlock()

	// If we haven't sent and processed a snapshot, diagd isn't ready.
	if w.LastSent.IsZero() || w.LastProcessed.IsZero() {
		return false
	}

	// We've sent and processed snapshots. If the last snapshot was sent before
	// the last snapshot was processed, we're good to go.
	if w.LastSent.Before(w.LastProcessed) {
		return true
	}

	// LastSent was after LastProcessed; we're still working on processing the
	// most recent snapshot. We'll say we're ready only as long as we're still
	// in the grace period.
	return w.withinGracePeriod()
}
