// Copyright 2019-2020 Datawire. All rights reserved.
//
// Copyright 2016 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package derrgroup provides synchronization, error propagation, and Context
// cancelation for groups of goroutines working on subtasks of a common task.
//
// derrgroup is a fork of golang.org/x/sync/errgroup commit
// 6e8e738ad208923de99951fe0b48239bfd864f28 (2020-06-04).
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

// A Group is a collection of goroutines working on subtasks that are part of
// the same overall task.
//
// A zero Group is valid and does not cancel on error.
type Group struct {
	cancel func()

	wg sync.WaitGroup

	listMu sync.RWMutex
	list   map[string]GoroutineState

	errOnce sync.Once
	err     error
}

// NewGroup returns a new Group.
//
// The provided 'cancel' function is called the first time a function passed to
// Go returns a non-nil error or the first time Wait returns, whichever occurs
// first.
func NewGroup(cancel func()) *Group {
	return &Group{cancel: cancel}
}

// Wait blocks until all function calls from the Go method have returned, then
// returns the first non-nil error (if any) from them.
func (g *Group) Wait() error {
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
