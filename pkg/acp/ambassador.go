// Copyright 2020 Datawire. All rights reserved.
//
// package acp contains stuff dealing with the Ambassador Control Plane as a whole.
//
// This is the AmbassadorWatcher, which is a class that can keep an eye on a running
// Ambassador as a whole, and tell you whether it's alive and ready, or not.
//
// THE STATE MACHINE AND THE GRACE PERIOD:
// When an Ambassador pod boots, Envoy is _not_ started at boot time: instead, we
// wait for the initial configuration to be generated and only then start Envoy (this
// is because we don't know what port to tell Envoy to listen on until after we've
// generated the initial configuration). However, Envoy can't come up instantly,
// either.
//
// To deal with this, there's a state machine in here. We start in envoyNotStarted
// state, move to envoyStarting when the first snapshot is processed, then move to
// envoyRunning when we successfully get stats back from Envoy. These states inform
// what, exactly, we demand to declare Ambassador alive, dead, ready, or not ready.
//
// If you like pictures instead of words:   envoyNotStarted
//                                                |
//                                                | (first snapshot is processed)
//                                                V
//                                          envoyStarting
//                                                |
//                                                | (we got stats from Envoy)
//                                                V
//                                          envoyRunning
//
// Envoy is currently given 30 seconds to come up after getting its initial
// configuration. This may be the wrong compromise: in practice, Envoy should come
// up _much_ faster than that, but the idea this code is more about providing a
// conservative failsafe than providing a finely-tuned hair trigger.
//
// TESTING HOOKS:
// Since time plays a role, you can use AmbassadorWatcher.SetFetchTime to change the
// function that the AmbassadorWatcher uses to fetch times. The default is time.Now.
//
// This hook is NOT meant for you to change the values on the fly in a running
// AmbassadorWatcher. Set it at instantiation if need be, then leave it alone. See
// ambassador_test.go for more.

package acp

import (
	"context"
	"fmt"
	"sync"
	"time"
)

type awState int

const (
	envoyNotStarted awState = iota
	envoyStarting
	envoyRunning
)

// AmbassadorWatcher encapsulates state and methods for keeping an eye on a running
// Ambassador, and deciding if it's healthy.
type AmbassadorWatcher struct {
	// This mutex protects access our watchers and our state, mostly as
	// a matter of rank paranoia.
	mutex sync.Mutex

	// How shall we fetch the current time?
	fetchTime timeFetcher

	// What's the current Envoy state?
	state awState

	// We encapsulate an EnvoyWatcher and a DiagdWatcher.
	ew *EnvoyWatcher
	dw *DiagdWatcher

	// At the point that the DiagdWatcher finishes processing the very first
	// snapshot, we have to hand the snapshot to Envoy and allow Envoy to start
	// up. This takes finite time, so we have to allow for that.
	GraceEnd time.Time
}

// NewAmbassadorWatcher creates a new AmbassadorWatcher, given a fetcher.
//
// Honestly, this is slightly pointless -- it's here for parallelism with the
// EnvoyWatcher and the DiagdWatcher.
func NewAmbassadorWatcher(ew *EnvoyWatcher, dw *DiagdWatcher) *AmbassadorWatcher {
	return &AmbassadorWatcher{
		// Default to using time.Now for time. This can be reset later.
		fetchTime: time.Now,
		state:     envoyNotStarted,
		ew:        ew,
		dw:        dw,
	}
}

// SetFetchTime will change the function we use to get the current time.
func (w *AmbassadorWatcher) SetFetchTime(fetchTime timeFetcher) {
	w.fetchTime = fetchTime
}

// FetchEnvoyReady will check whether Envoy's statistics are fetchable.
func (w *AmbassadorWatcher) FetchEnvoyReady(ctx context.Context) {
	w.mutex.Lock()
	defer w.mutex.Unlock()

	w.ew.FetchEnvoyReady(ctx)
}

// NoteSnapshotSent will note that a snapshot has been sent.
func (w *AmbassadorWatcher) NoteSnapshotSent() {
	w.mutex.Lock()
	defer w.mutex.Unlock()

	w.dw.NoteSnapshotSent()
}

// NoteSnapshotProcessed will note that a snapshot has been processed.
func (w *AmbassadorWatcher) NoteSnapshotProcessed() {
	w.mutex.Lock()
	defer w.mutex.Unlock()

	w.dw.NoteSnapshotProcessed()

	// Is this is the very first time we've processed a snapshot?
	if w.state == envoyNotStarted {
		// Yes, it is. Note that we're now waiting for Envoy to start...
		w.state = envoyStarting

		// ...and give Envoy 30 seconds to come up.
		w.GraceEnd = w.fetchTime().Add(30 * time.Second)
	}
}

// IsAlive returns true IFF the Ambassador as a whole can be considered alive.
func (w *AmbassadorWatcher) IsAlive() bool {
	w.mutex.Lock()
	defer w.mutex.Unlock()

	// First things first: if diagd isn't alive, Ambassador as a whole is
	// clearly not alive.

	if !w.dw.IsAlive() {
		return false
	}

	// OK, diagd is alive. We need to look at our current state to figure
	// out how we'll check Envoy.

	switch w.state {
	case envoyNotStarted:
		// We haven't even tried to start Envoy yet, so we're good to go with
		// just diagd being alive.
		return true

	case envoyStarting:
		// We're waiting for Envoy to start. Has it?
		if w.ew.IsAlive() {
			// Yes. Remember that it's running...
			w.state = envoyRunning

			// ...and then we're good to go.
			return true
		}

		// It's not yet running. Return true IFF we're still within the grace period.
		return w.fetchTime().Before(w.GraceEnd)

	case envoyRunning:
		// Envoy is already running, so check to make sure that it's still alive.
		return w.ew.IsAlive()

	default:
		// This is "impossible": w.state isn't exported, and it's deliberately
		// typed as awState to catch someone trying to assign a random integer to
		// it. However, I guess someone could conceivably assign something new to
		// it without updating this code, so we test for it.
		panic(fmt.Sprintf("AmbassadorWatcher.state enum has unknown value %d", w.state))
	}
}

// IsReady returns true IFF the Ambassador as a whole can be considered ready.
func (w *AmbassadorWatcher) IsReady() bool {
	w.mutex.Lock()
	defer w.mutex.Unlock()

	// This is much simpler that IsAlive. Ambassador is ready IFF both diagd and
	// Envoy are ready; that's all there is to it.

	return w.dw.IsReady() && w.ew.IsReady()
}
