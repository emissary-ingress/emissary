// Package dgroup provides tools for managing groups of goroutines.
//
// The main part of this is Group, but the naming utilities may be
// useful outside of that.
//
// dgroup should be the goroutine-group-management abstraction used by
// all new code at Ambassador Labs.  The principle remaining
// limitation (at least when compared to the *other* Ambassador Labs
// "goroutine group" library that shall-not-be-named) is that dgroup
// does not have a notion of dependencies.  Not having a notion of
// dependencies implies a few things:
//
//  - it does not have a notion of "readiness", as it doesn't have any
//    dependents that would block on a goroutine becoming ready
//  - it launches worker goroutines right away when you call .Go(), as
//    it doesn't have any dependencies that would block the worker
//    from starting
//
// So, if you need to enforce ordering requirements during goroutine
// startup and shutdown, then (for now, anyway) you'll need to
// implement that separately on top of this library.
package dgroup

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"runtime/pprof"
	"sort"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/pkg/errors"

	"github.com/datawire/dlib/dcontext"
	"github.com/datawire/dlib/derrgroup"
	"github.com/datawire/dlib/derror"
	"github.com/datawire/dlib/dlog"
)

// A Group is a collection of goroutines working on subtasks that are
// part of the same overall task.  Compared to a minimum-viable
// goroutine-group abstraction (such as stdlib "sync.WaitGroup" or the
// bog-standard "golang.org/x/sync/errgroup.Group"), the things that
// make dgroup attractive are:
//
//  - (optionally) handles SIGINT and SIGTERM
//  - (configurable) manages Context for you
//  - (optionally) adds hard/soft cancellation
//  - (optionally) does panic recovery
//  - (optionally) does some minimal logging
//  - (optionally) adds configurable shutdown timeouts
//  - a concept of goroutine names
//  - adds a way to call to the parent group, making it possible to
//    launch a "sibling" goroutine
//
// A zero Group is NOT valid; a Group must be created with NewGroup.
//
// A Group is suitable for use at the top-level of a program
// (especially if signal handling is enabled for the Group), and is
// also suitable to be nested somewhere deep inside of an application
// (but signal handling should probably be disabled for that use).
type Group struct {
	cfg     GroupConfig
	baseCtx context.Context

	shutdownTimedOut chan struct{}
	waitFinished     chan struct{}
	hardCancel       context.CancelFunc

	workers     *derrgroup.Group
	supervisors sync.WaitGroup
}

func logGoroutineStatuses(
	ctx context.Context,
	heading string,
	printf func(ctx context.Context, format string, args ...interface{}),
	list map[string]derrgroup.GoroutineState,
) {
	printf(ctx, "  %s:", heading)
	names := make([]string, 0, len(list))
	nameWidth := 0
	for name := range list {
		names = append(names, name)
		if len(name) > nameWidth {
			nameWidth = len(name)
		}
	}
	sort.Strings(names)
	for _, name := range names {
		printf(ctx, "    %-*s: %s", nameWidth, name, list[name])
	}
}

var stacktraceForTesting string

func logGoroutineTraces(
	ctx context.Context,
	heading string,
	printf func(ctx context.Context, format string, args ...interface{}),
) {
	stacktrace := new(strings.Builder)
	if stacktraceForTesting != "" {
		stacktrace.WriteString(stacktraceForTesting)
	} else {
		p := pprof.Lookup("goroutine")
		if p == nil {
			return
		}
		if err := p.WriteTo(stacktrace, 2); err != nil {
			return
		}
	}
	printf(ctx, "  %s:", heading)
	for _, line := range strings.Split(strings.TrimSpace(stacktrace.String()), "\n") {
		printf(ctx, "    %s", line)
	}
}

// GroupConfig is a readable way of setting the configuration options
// for NewGroup.
//
// A zero GroupConfig (`dgroup.GroupConfig{}`) should be sane
// defaults.  Because signal handling should only be enabled for the
// outermost group, it is off by default.
//
// TODO(lukeshu): Consider enabling timeouts by default?
type GroupConfig struct {
	// EnableWithSoftness says whether it should call
	// dcontext.WithSoftness() on the Context passed to NewGroup.
	// This should probably NOT be set for a Context that is
	// already soft.  However, this must be set for features that
	// require separate hard/soft cancellation, such as signal
	// handling.  If any of those features are enabled, then it
	// will force EnableWithSoftness to be set.
	EnableWithSoftness   bool
	EnableSignalHandling bool // implies EnableWithSoftness

	// Normally a worker exiting with an error triggers other
	// goroutines to shutdown.  Setting ShutdownOnNonError causes
	// a shutdown to be triggered whenever a goroutine exits, even
	// if it exits without error.
	ShutdownOnNonError bool

	// SoftShutdownTimeout is how long after a soft shutdown is
	// triggered to wait before triggering a hard shutdown.  A
	// zero value means to not trigger a hard shutdown after a
	// soft shutdown.
	//
	// SoftShutdownTimeout implies EnableWithSoftness because
	// otherwise there would be no way of triggering the
	// subsequent hard shutdown.
	SoftShutdownTimeout time.Duration
	// HardShutdownTimeout is how long after a hard shutdown is
	// triggered to wait before forcing Wait() to return early.  A
	// zero value means to not force Wait() to return early.
	HardShutdownTimeout time.Duration

	DisablePanicRecovery bool
	DisableLogging       bool

	WorkerContext func(ctx context.Context, name string) context.Context
}

// NewGroup returns a new Group.
func NewGroup(ctx context.Context, cfg GroupConfig) *Group {
	cfg.EnableWithSoftness = cfg.EnableWithSoftness || cfg.EnableSignalHandling || (cfg.SoftShutdownTimeout > 0)

	ctx, hardCancel := context.WithCancel(ctx)
	var softCancel context.CancelFunc
	if cfg.EnableWithSoftness {
		ctx = dcontext.WithSoftness(ctx)
		ctx, softCancel = context.WithCancel(ctx)
	} else {
		softCancel = hardCancel
	}

	g := &Group{
		cfg: cfg,
		//baseCtx: gets set below,

		shutdownTimedOut: make(chan struct{}),
		waitFinished:     make(chan struct{}),
		hardCancel:       hardCancel,

		workers: derrgroup.NewGroup(softCancel, cfg.ShutdownOnNonError),
		//supervisors: zero value is fine; doesn't need initialize,
	}
	g.baseCtx = context.WithValue(ctx, groupKey{}, g)

	g.launchSupervisors()

	return g
}

// launchSupervisors launches the various "internal" / "supervisor" /
// "helper" goroutines that aren't of concern to the caller of dgroup,
// but are internal to implementing dgroup's various features.
func (g *Group) launchSupervisors() {
	if !g.cfg.DisableLogging {
		g.goSupervisor("shutdown_logger", func(ctx context.Context) {
			// We should be as specific with logging as possible.

			// Wait for shutdown to be initiated (or for everything to quit on
			// its own).
			select {
			case <-g.waitFinished:
			case <-ctx.Done():
			}
			// Check whether <-ctx.Done() happened; we do this separately
			// after-the-fact (instead of in the select case) because it's
			// possible that they both happen, and if they both happen then
			// `select` will choose one arbitrarily, but we still need to do
			// this if the `select` chooses <-g.waitFinished.
			if ctx.Err() == nil {
				// Only <-g.waitFinished happened;
				// we won't have anything to log.
				return
			}
			if dcontext.HardContext(ctx) == ctx {
				// No hard/soft distinction
				dlog.Infoln(ctx, "shutting down...")
				return
			} else {
				// There is a hard/soft distinction; check whether it was
				// a hard or soft shutdown that was triggered...
				if dcontext.HardContext(ctx).Err() != nil {
					// It was a hard; log that...
					dlog.Infoln(ctx, "shutting down (not-so-gracefully)...")
					// ...then we're done
					return
				} else {
					// It was soft; log that...
					dlog.Infoln(ctx, "shutting down (gracefully)...")
					// ...now we need to do the same thing again to
					// log when hard-shutdown is initiated.
					select {
					case <-g.waitFinished:
					case <-dcontext.HardContext(ctx).Done():
					}
					if dcontext.HardContext(ctx).Err() == nil {
						// Only <-g.waitFinished happened;
						// we won't have anything to log.
						return
					}
					dlog.Infoln(ctx, "shutting down (not-so-gracefully)...")
				}
			}
		})
	}

	if (g.cfg.SoftShutdownTimeout > 0) || (g.cfg.HardShutdownTimeout > 0) {
		g.goSupervisor("timeout_watchdog", func(ctx context.Context) {
			if g.cfg.SoftShutdownTimeout > 0 {
				select {
				case <-g.waitFinished:
					// nothing to do
				case <-ctx.Done():
					// soft-shutdown initiated, start the soft-shutdown timeout-clock
					select {
					case <-g.waitFinished:
						// nothing to do, it finished within the timeout
					case <-dcontext.HardContext(ctx).Done():
						// nothing to do, something else went ahead and upgraded
						// this to a hard-shutdown
					case <-time.After(g.cfg.SoftShutdownTimeout):
						// it didn't finish within the timeout,
						// upgrade to a hard-shutdown
						g.hardCancel()
					}
				}
			}
			if g.cfg.HardShutdownTimeout > 0 {
				select {
				case <-g.waitFinished:
					// nothing to do
				case <-dcontext.HardContext(ctx).Done():
					// hard-shutdown initiated, start the hard-shutdown timeout-clock
					select {
					case <-g.waitFinished:
						// nothing to do, it finished within the timeout
					case <-time.After(g.cfg.HardShutdownTimeout):
						close(g.shutdownTimedOut)
					}
				}
			}
		})
	}

	if g.cfg.EnableSignalHandling {
		sigs := make(chan os.Signal, 1)
		signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
		g.goSupervisor("signal_handler", func(ctx context.Context) {
			<-g.waitFinished
			signal.Stop(sigs)
			close(sigs)
		})
		g.goSupervisor("signal_handler", func(ctx context.Context) {
			i := 0
			for sig := range sigs {
				ctx := WithGoroutineName(ctx, fmt.Sprintf(":%d", i))
				i++

				// Specifically use fmt.Errorf instead of errors.Errorf here, to avoid including a
				// stacktrace with the error, these are "expected" errors, and including stacktraces for
				// them in the group's exit logging would just be noise.
				if ctx.Err() == nil {
					err := fmt.Errorf("received signal %v (triggering graceful shutdown)", sig)

					g.goWorkerCtx(ctx, func(_ context.Context) error {
						return err
					})
					<-ctx.Done()

				} else if dcontext.HardContext(ctx).Err() == nil {
					err := fmt.Errorf("received signal %v (graceful shutdown already triggered; triggering not-so-graceful shutdown)", sig)

					if !g.cfg.DisableLogging {
						dlog.Errorln(ctx, err)
						logGoroutineStatuses(ctx, "goroutine statuses", dlog.Errorf, g.List())
					}
					g.hardCancel()

				} else {
					err := fmt.Errorf("received signal %v (not-so-graceful shutdown already triggered)", sig)

					if !g.cfg.DisableLogging {
						dlog.Errorln(ctx, err)
						logGoroutineStatuses(ctx, "goroutine statuses", dlog.Errorf, g.List())
						logGoroutineTraces(ctx, "goroutine stack traces", dlog.Errorf)
					}
				}
			}
		})
	}
}

// Go calls the given function in a new named-worker-goroutine.
//
// Cancellation of the Context should trigger a graceful shutdown.
// Cancellation of the dcontext.HardContext(ctx) of it should trigger
// a not-so-graceful shutdown.
//
// A worker may access its parent group by calling ParentGroup on its
// Context.
func (g *Group) Go(name string, fn func(ctx context.Context) error) {
	g.goWorker(name, fn)
}

// goWorker launches a worker goroutine for the user of dgroup.
func (g *Group) goWorker(name string, fn func(ctx context.Context) error) {
	ctx := WithGoroutineName(g.baseCtx, "/"+name)
	if g.cfg.WorkerContext != nil {
		ctx = g.cfg.WorkerContext(ctx, name)
	}
	g.goWorkerCtx(ctx, fn)
}

// goWorkerCtx() is like goWorker(), except it takes an
// already-created context.
func (g *Group) goWorkerCtx(ctx context.Context, fn func(ctx context.Context) error) {
	g.workers.Go(getGoroutineName(ctx), func() (err error) {
		defer func() {
			if !g.cfg.DisablePanicRecovery {
				if _err := derror.PanicToError(recover()); _err != nil {
					err = _err
				}
			}
			if !g.cfg.DisableLogging {
				if err == nil {
					dlog.Debugf(ctx, "goroutine %q exited without error", getGoroutineName(ctx))
				} else {
					// Use %+v instead of %v to include the stacktrace (if there is one).  In
					// particular, if the above panic recovery tripped, then we really don't want to
					// throw away the stacktrace.
					dlog.Errorf(ctx, "goroutine %q exited with error: %+v", getGoroutineName(ctx), err)
				}
			}
		}()

		return fn(ctx)
	})
}

// goSupervisor launches an "internal" / "supervisor" / "helper"
// goroutine that isn't of concern to the caller of dgroup, but is
// internal to implementing one of dgroup's features.  Put another
// way: they are "systems-logic" goroutines, not "business-logic"
// goroutines.
//
// Compared to normal user-provided "worker" goroutines, these
// "supervisor" goroutines have a few important differences and
// additional requirements:
//
//  - They MUST monitor the g.waitFinished channel, and MUST finish
//    quickly after that channel is closed.
//  - They MUST not panic, as we don't bother to set up panic recovery
//    for them.
//  - The cfg.WorkerContext() callback is not called.
//  - Being a "systems" thing, they must be robust and CANNOT fail; so
//    they don't get to return an error.
func (g *Group) goSupervisor(name string, fn func(ctx context.Context)) {
	ctx := WithGoroutineName(g.baseCtx, ":"+name)
	g.goSupervisorCtx(ctx, fn)
}

// goSupervisorCtx() is like goSupervisor(), except it takes an
// already-created context.
func (g *Group) goSupervisorCtx(ctx context.Context, fn func(ctx context.Context)) {
	g.supervisors.Add(1)
	go func() {
		defer g.supervisors.Done()
		fn(ctx)
	}()
}

// Wait for all goroutines in the group to finish, and return returns
// an error if any of the workers errored or timed out.
//
// Once the group has initiated hard-shutdown (either soft-shutdown
// was initiated then timed out, a 2nd shutdown signal was received,
// or the parent context is <-Done()), Wait will return within the
// HardShutdownTimeout passed to NewGroup.  If a poorly-behaved
// goroutine is still running at the end of that time, it is left
// running, and an error is returned.
func (g *Group) Wait() error {
	// 1. Wait for the worker goroutines to finish (or time out)
	shutdownCompleted := make(chan error)
	go func() {
		shutdownCompleted <- g.workers.Wait()
		close(shutdownCompleted)
	}()
	var ret error
	var timedOut bool
	select {
	case <-g.shutdownTimedOut:
		ret = errors.Errorf("failed to shut down within the %v shutdown timeout; some goroutines are left running", g.cfg.HardShutdownTimeout)
		timedOut = true
	case ret = <-shutdownCompleted:
	}

	// 2. Quit the supervisor goroutines
	close(g.waitFinished)
	g.supervisors.Wait()

	// 3. Belt-and-suspenders: Make sure that anything branched
	// from our Context observes that this group is no longer
	// running.
	g.hardCancel()

	// 4. Log the result and return
	if ret != nil && !g.cfg.DisableLogging {
		ctx := WithGoroutineName(g.baseCtx, ":shutdown_status")
		logGoroutineStatuses(ctx, "final goroutine statuses", dlog.Infof, g.List())
		if timedOut {
			logGoroutineTraces(ctx, "final goroutine stack traces", dlog.Errorf)
		}
	}
	return ret
}

// List returns a listing of all goroutines launched with .Go().
func (g *Group) List() map[string]derrgroup.GoroutineState {
	return g.workers.List()
}

type groupKey struct{}

// ParentGroup returns the Group that manages this goroutine/Context.
// If the Context is not managed by a Group, then nil is returned.
// The principle use of ParentGroup is to launch a sibling goroutine
// in the group.
func ParentGroup(ctx context.Context) *Group {
	group := ctx.Value(groupKey{})
	if group == nil {
		return nil
	}
	return group.(*Group)
}
