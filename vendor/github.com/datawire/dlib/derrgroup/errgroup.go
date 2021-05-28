// Copyright 2019-2020 Datawire. All rights reserved.
//
// Copyright 2016 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package derrgroup is a low-level group abstraction; providing
// synchronization, error propagation, and cancellation callback for
// groups of goroutines working on subtasks of a common task.
//
// derrgroup is a fork of golang.org/x/sync/errgroup commit
// 6e8e738ad208923de99951fe0b48239bfd864f28 (2020-06-04).  It is
// forked to provide only things that cannot reasonably be implemented
// on top of itself; it is impossible to add goroutine enumeration on
// top of errgroup without duplicating and doubling up on all of
// errgroup's synchronization/locking.  Anything that can reasonably
// be implemented *on top of* derrgroup is not included in derrgroup:
//  - Managing `context.Contexts`s (this is something that errgroup
//    kind of does, but derrgroup ripped out, because it can trivially
//    be implemented on top of derrgroup)
//  - Signal handling
//  - Logging
//  - Hard/soft cancellation
//  - Having `Wait()` timeout on a shutdown that takes too long
// Those are all good and useful things to have.  But they should be
// implemented in a layer *on top of* derrgroup. "derrgroup.Group" was
// originally called "llGroup" for "low-level group"; it is
// intentionally low-level in order the be a clean primitive for other
// things to build on top of.
//
// Right now, there have been at least 3 Go implementations of "group"
// functionality in use at Datawire at the same time (pkg/dgroup and
// pkg/supervisor are the two still in use), which each offer some
// subset of the above.  derrgroup offers to them a common robust
// base.  If you're writing new application code, you should use one
// of those, and not use derrgroup directly.  If you're writing a new
// "group" abstraction, you should use derrgroup instead of
// implementing your own locking/synchronization.
package derrgroup

import (
	"sync"

	"github.com/pkg/errors"
)

type GoroutineState int

const (
	GoroutineRunning GoroutineState = iota
	GoroutineExited
	GoroutineErrored
)

func (s GoroutineState) String() string {
	switch s {
	case GoroutineRunning:
		return "running"
	case GoroutineExited:
		return "exited without error"
	case GoroutineErrored:
		return "exited with error"
	default:
		panic(errors.Errorf("invalid GoroutineState = %d", s))
	}
}

// A Group is a collection of goroutines working on subtasks that are part of
// the same overall task.
//
// A zero Group is valid and does not cancel on error.
type Group struct {
	cancel           func()
	cancelOnNonError bool

	wg sync.WaitGroup

	listMu sync.RWMutex
	list   map[string]GoroutineState

	errOnce sync.Once
	err     error
}

// NewGroup returns a new Group.
//
// The provided 'cancel' function is called the first time a function passed to
// Go returns a non-nil error.
func NewGroup(cancel func(), cancelOnNonError bool) *Group {
	return &Group{
		cancel:           cancel,
		cancelOnNonError: cancelOnNonError,
	}
}

// Wait blocks until all function calls from the Go method have returned, then
// returns the first non-nil error (if any) from them.
func (g *Group) Wait() error {
	g.wg.Wait()
	return g.err
}

// Go calls the given function in a new goroutine.
//
// The first call to return a non-nil error cancels the group; its error will be
// returned by Wait.
func (g *Group) Go(name string, f func() error) {
	g.listMu.Lock()
	if g.list == nil {
		g.list = make(map[string]GoroutineState)
	}
	if _, exists := g.list[name]; exists {
		g.wg.Add(1)
		g.listMu.Unlock()
		go func() {
			g.errOnce.Do(func() {
				g.err = errors.Errorf("a goroutine with name %q already exists", name)
				if g.cancel != nil {
					g.cancel()
				}
			})
			g.wg.Done()
		}()
		return
	}
	g.list[name] = GoroutineRunning
	g.wg.Add(1)
	g.listMu.Unlock()

	go func() {
		exitState := GoroutineExited
		if err := f(); err != nil {
			exitState = GoroutineErrored
			g.errOnce.Do(func() {
				g.err = err
				if g.cancel != nil {
					g.cancel()
				}
			})
		} else if g.cancelOnNonError {
			g.cancel()
		}
		g.listMu.Lock()
		if g.list == nil {
			g.list = make(map[string]GoroutineState)
		}
		g.list[name] = exitState
		g.wg.Done()
		g.listMu.Unlock()
	}()
}

// List returns a listing of all goroutines launched with Go.
func (g *Group) List() map[string]GoroutineState {
	g.listMu.RLock()
	defer g.listMu.RUnlock()

	ret := make(map[string]GoroutineState, len(g.list))
	for k, v := range g.list {
		ret[k] = v
	}

	return ret
}
