package entrypoint

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/datawire/ambassador/pkg/derrgroup"
	"github.com/datawire/ambassador/pkg/errutil"
)

// Group manages a group group of related goroutines.
//
// TODO(lukeshu): Merge this with cmd/amb-sidecar/group.
type Group struct {
	inner *derrgroup.Group

	// To pass to the worker goroutines
	ctx context.Context

	// For dealing with shutdown
	grace            time.Duration
	cancel           func()
	shutdownTimedOut chan struct{}
}

// NewGroup returns a manager for group of related goroutines:
//
//  - If one dies, they all die.
//  - If one is poorly behaved and does not die within `grace` of its
//    Context being canceled, then Wait() returns early, and indicates
//    the poorly behaved goroutine.  Because it is not possible to
//    kill a goroutine, the poorly behaved goroutine will still be
//    running when Wait() returns.
//  - Capture all background errors so it is easy to ensure they are
//    repeated when the program terminates and not swallowed in log
//    output.
func NewGroup(parent context.Context, grace time.Duration) *Group {
	ctx, cancel := context.WithCancel(parent)
	ret := &Group{
		ctx:              ctx,
		grace:            grace,
		shutdownTimedOut: make(chan struct{}),
		inner:            derrgroup.NewGroup(cancel),
	}
	go func() {
		<-ctx.Done()
		time.Sleep(grace)
		close(ret.shutdownTimedOut)
	}()
	return ret
}

// Launch a goroutine as part of the group.
func (g *Group) Go(name string, f func(context.Context)) {
	g.inner.Go(name, func() (err error) {
		// exit bookeeping:
		//  1. Log that we exited.
		//  2. Cancel the context so others know to exit.
		defer func() {
			err = errutil.PanicToError(recover())
			if err == nil {
				log.Printf("EXIT %s normal", name)
			} else {
				log.Printf("EXIT %s panic: %v", name, err)
			}
			g.cancel() // trigger a shutdown whether or not there was an error
		}()
		f(g.ctx)
		return
	})
}

// Wait for all goroutines in the group to finish, and return a map of
// any errors.  No news is good news.  If the map does not contain an
// entry for the goroutine, then it exited normally.
//
// Once the group has initiated shutdown (either one of the goroutines
// has exited, or the parent context has been canceled), Wait will
// return within the `grace` period passed to NewGroup.  If a
// poorly-behaved goroutine is still running at the end of that time,
// it is left running, and it is indicated as "running, did not exit
// in time ({{grace}} shutdown grace)" in the returned map.
func (g *Group) Wait() map[string]string {
	shutdownCompleted := make(chan struct{})
	go func() {
		g.inner.Wait()
		close(shutdownCompleted)
	}()
	select {
	case <-g.shutdownTimedOut:
	case <-shutdownCompleted:
	}
	rawList := g.inner.List()
	formattedList := make(map[string]string, len(rawList))
	for name, state := range rawList {
		switch state {
		case derrgroup.GoroutineRunning:
			formattedList[name] = fmt.Sprintf("%s, did not exit in time (%v shutdown grace)", state, g.grace)
		case derrgroup.GoroutineErrored:
			formattedList[name] = state.String()
		case derrgroup.GoroutineExited:
			// leave it out of the result
		}
	}
	return formattedList
}
