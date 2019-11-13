// Copyright 2016 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package runner

import (
	"sync"
)

// A llGroup is a collection of goroutines working on subtasks that are part of
// the same overall task.
//
// A zero llGroup is valid and does not cancel on error.
type llGroup struct {
	cancel func()

	wg sync.WaitGroup

	errOnce sync.Once
	err     error
}

// newLLGroup returns a new llGroup.
//
// The provided 'cancel' function is called the first time a function passed to
// Go returns a non-nil error or the first time Wait returns, whichever occurs
// first.
func newLLGroup(cancel func()) *llGroup {
	return &llGroup{cancel: cancel}
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
func (g *llGroup) Go(f func() error) {
	g.wg.Add(1)

	go func() {
		defer g.wg.Done()

		if err := f(); err != nil {
			g.errOnce.Do(func() {
				g.err = err
				if g.cancel != nil {
					g.cancel()
				}
			})
		}
	}()
}
