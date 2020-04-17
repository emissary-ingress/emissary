// Copyright 2016 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package group

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

// A llGroup is a collection of goroutines working on subtasks that are part of
// the same overall task.
//
// A zero llGroup is valid and does not cancel on error.
type llGroup struct {
	cancel func()

	wg sync.WaitGroup

	listMu sync.RWMutex
	list   map[string]GoroutineState

	errOnce sync.Once
	err     error
}

// newLLGroup returns a new llGroup.
//
// The provided 'cancel' function is called the first time a function passed to
// Go returns a non-nil error or the first time Wait returns, whichever occurs
// first.
func newLLGroup(cancel func()) *llGroup {
	return &llGroup{
		cancel: cancel,
		list:   make(map[string]GoroutineState),
	}
}

// Wait blocks until all function calls from the Go method have returned, then
// returns the first non-nil error (if any) from them.
func (g *llGroup) Wait() error {
	g.wg.Wait()
	if g.cancel != nil {
		g.cancel()
	}
	return g.err
}

// Go calls the given function in a new goroutine.
//
// The first call to return a non-nil error cancels the group; its error will be
// returned by Wait.
func (g *llGroup) Go(name string, f func() error) {
	g.listMu.Lock()
	if _, exists := g.list[name]; exists {
		g.wg.Add(1)
		g.listMu.Unlock()
		go func() {
			g.errOnce.Do(func() {
				g.err = errors.Errorf("a goroutine with name %q already exists", name)
				g.cancel()
			})
			g.wg.Done()
		}()
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
				g.cancel()
			})
		}
		g.listMu.Lock()
		g.list[name] = exitState
		g.wg.Done()
		g.listMu.Unlock()
	}()
}

// List returns a listing of all goroutines launched with Go.
func (g *llGroup) List() map[string]GoroutineState {
	g.listMu.RLock()
	defer g.listMu.RUnlock()

	ret := make(map[string]GoroutineState, len(g.list))
	for k, v := range g.list {
		ret[k] = v
	}

	return ret
}
