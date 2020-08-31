package entrypoint

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"
)

// XXX: should we replace with Luke's stuff?
// Manage a group of related goroutines:
//  - if one dies, we all die
//    + even if one of us is poorly behaved
//  - capture all background errors so it is easy to ensure they are repeated when the program
//    terminates and not swallowed in log output
//  - always report poorly behaved goroutines that refuse to exit
type Group struct {
	// Parent context
	ctx context.Context

	// How long we should wait for other routines to shutdown.
	grace time.Duration

	cancel context.CancelFunc
	once   sync.Once
	wg     sync.WaitGroup

	// The "do" mutex, protects running, paniced, and all invocations of "do".
	mutex   sync.Mutex
	running map[string]bool
	paniced map[string]interface{}
}

func NewGroup(parent context.Context, grace time.Duration) *Group {
	ctx, cancel := context.WithCancel(parent)
	return &Group{
		ctx:     ctx,
		grace:   grace,
		cancel:  cancel,
		running: map[string]bool{},
		paniced: map[string]interface{}{},
	}
}

// Launch a goroutine as part of the group.
func (g *Group) Go(name string, f func(context.Context)) {
	g.do(func() {
		_, ok := g.running[name]
		if ok {
			panic(fmt.Sprintf("duplicate goroutine name: %s", name))
		}
		g.running[name] = true
	})

	g.wg.Add(1)
	go func() {
		// exit bookeeping:
		//  1. updated running and paniced maps
		//  2. call g.wg.Done() to signal we have exited
		//  3. cancel the context so others know to exit
		//  4. if we are the first to exit, start a watchdog in case others don't exit
		defer func() {
			err := recover()
			if err == nil {
				g.do(func() { g.running[name] = false })
				log.Printf("EXIT %s normal", name)
			} else {
				g.do(func() { g.running[name] = false; g.paniced[name] = err })
				log.Printf("EXIT %s panic: %v", name, err)
			}
			g.wg.Done()
			g.cancel()
			g.once.Do(func() { go g.watchdog() })
		}()
		f(g.ctx)
	}()
}

// This is the watchdog, sleep for the grace period and then call g.wg.Done() on behalf of any
// goroutines that have not yet exited and fill in an error.
func (g *Group) watchdog() {
	time.Sleep(g.grace)
	g.do(func() {
		for _, isRunning := range g.running {
			if isRunning {
				g.wg.Done()
			}
		}
	})
}

// Wait for all goroutines in the group to finish, and return a map of any errors. No news is good
// news. If the map does not contain an entry for the goroutine, then it exited normally.
func (g *Group) Wait() map[string]interface{} {
	g.wg.Wait()
	result := map[string]interface{}{}
	g.do(func() {
		for k, v := range g.running {
			if v {
				result[k] = fmt.Errorf("%s did not exit in time (%s shutdown grace)", k, g.grace.String())
			}
		}
		for k, v := range g.paniced {
			result[k] = v
		}
	})
	return result
}

// Do something while holding the group mutex.
func (g *Group) do(f func()) {
	g.mutex.Lock()
	defer g.mutex.Unlock()
	f()
}
