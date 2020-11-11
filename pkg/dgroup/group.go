// Package dgroup provides tools for managing groups of goroutines.
//
// The main part of this is Group, but the naming utilities may be
// useful outside of that.
//
// At this point, the limitation of dgroup when compared to supervisor
// is that dgroup does not have a notion of readiness, and does not
// have a notion of dependencies.
package dgroup

import (
	"context"
	"os"
	"os/signal"
	"runtime/pprof"
	"sort"
	"strings"
	"syscall"
	"time"

	"github.com/pkg/errors"

	"github.com/datawire/ambassador/pkg/dcontext"
	"github.com/datawire/ambassador/pkg/derrgroup"
	"github.com/datawire/ambassador/pkg/dlog"
	"github.com/datawire/ambassador/pkg/errutil"
)

// Group is a wrapper around
// github.com/datawire/ambassador/pkg/derrgroup.Group that:
//  - (optionally) handles SIGINT and SIGTERM
//  - (configurable) manages Context for you
//  - (optionally) adds hard/soft cancellation
//  - (optionally) does panic recovery
//  - (optionally) does some minimal logging
//  - (optionally) adds configurable shutdown timeouts
//  - adds a way to call to the parent group
//
// A zero Group is NOT valid; a Group must be created with NewGroup.
type Group struct {
	cfg              GroupConfig
	baseCtx          context.Context
	shutdownTimedOut chan struct{}
	hardCancel       context.CancelFunc
	inner            *derrgroup.Group
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

func logGoroutineTraces(
	ctx context.Context,
	heading string,
	printf func(ctx context.Context, format string, args ...interface{}),
) {
	p := pprof.Lookup("goroutine")
	if p == nil {
		return
	}
	stacktrace := new(strings.Builder)
	if err := p.WriteTo(stacktrace, 2); err != nil {
		return
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
		cfg:              cfg,
		baseCtx:          ctx,
		shutdownTimedOut: make(chan struct{}),
		hardCancel:       hardCancel,
		inner:            derrgroup.NewGroup(softCancel, cfg.ShutdownOnNonError),
	}

	if !g.cfg.DisableLogging {
		g.Go(":shutdown_logger", func(ctx context.Context) error {
			<-ctx.Done()
			// log that a shutdown has been triggered
			// be as specific with the logging as possible
			if dcontext.HardContext(ctx) == ctx {
				dlog.Infoln(ctx, "shutting down...")
			} else {
				select {
				case <-dcontext.HardContext(ctx).Done():
					dlog.Infoln(ctx, "shutting down (not-so-gracefully)...")
				default:
					dlog.Infoln(ctx, "shutting down (gracefully)...")
					<-dcontext.HardContext(ctx).Done()
					dlog.Infoln(ctx, "shutting down (not-so-gracefully)...")
				}
			}
			return nil
		})
	}

	if (g.cfg.SoftShutdownTimeout > 0) || (g.cfg.HardShutdownTimeout > 0) {
		g.Go(":watchdog", func(ctx context.Context) error {
			if g.cfg.SoftShutdownTimeout > 0 {
				<-ctx.Done()
				select {
				case <-dcontext.HardContext(ctx).Done():
					// nothing to do, it finished within the timeout
				case <-time.After(g.cfg.SoftShutdownTimeout):
					hardCancel()
				}
			}
			if g.cfg.HardShutdownTimeout > 0 {
				<-dcontext.HardContext(ctx).Done()
				go func() {
					time.Sleep(g.cfg.HardShutdownTimeout)
					close(g.shutdownTimedOut)
				}()
			}
			return nil
		})
	}

	if g.cfg.EnableSignalHandling {
		g.Go(":signal_handler", func(ctx context.Context) error {
			sigs := make(chan os.Signal, 1)
			signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

			defer func() {
				// If we receive another signal after
				// graceful-shutdown, we should trigger a
				// not-so-graceful shutdown.
				go func() {
					select {
					case sig := <-sigs:
						if !g.cfg.DisableLogging {
							dlog.Errorln(ctx, errors.Errorf("received signal %v (graceful shutdown already triggered; triggering not-so-graceful shutdown)", sig))
							logGoroutineStatuses(ctx, "goroutine statuses", dlog.Errorf, g.List())
						}
						hardCancel()
					case <-dcontext.HardContext(ctx).Done():
					}
					// keep logging signals and draining 'sigs'--don't let 'sigs' block
					for sig := range sigs {
						if !g.cfg.DisableLogging {
							dlog.Errorln(ctx, errors.Errorf("received signal %v (not-so-graceful shutdown already triggered)", sig))
							logGoroutineStatuses(ctx, "goroutine statuses", dlog.Errorf, g.List())
							logGoroutineTraces(ctx, "goroutine stack traces", dlog.Errorf)
						}
					}
				}()
			}()

			select {
			case sig := <-sigs:
				return errors.Errorf("received signal %v (first signal; triggering graceful shutdown)", sig)
			case <-ctx.Done():
				return nil
			}
		})
	}

	return g
}

// Go wraps derrgroup.Group.Go().
//
// Cancellation of the Context should trigger a graceful shutdown.
// Cancellation of the dcontext.HardContext(ctx) of it should trigger
// a not-so-graceful shutdown.
//
// A worker may access its parent group by calling ParentGroup on its
// Context.
func (g *Group) Go(name string, fn func(ctx context.Context) error) {
	ctx := g.baseCtx
	ctx = WithGoroutineName(ctx, "/"+name)
	ctx = context.WithValue(ctx, groupKey{}, g)
	if g.cfg.WorkerContext != nil {
		ctx = g.cfg.WorkerContext(ctx, name)
	}

	g.inner.Go(getGoroutineName(ctx), func() (err error) {

		defer func() {
			if !g.cfg.DisablePanicRecovery {
				if _err := errutil.PanicToError(recover()); _err != nil {
					err = _err
				}
			}
			if !g.cfg.DisableLogging {
				if err == nil {
					dlog.Debugf(ctx, "goroutine %q exited without error", getGoroutineName(ctx))
				} else {
					dlog.Errorf(ctx, "goroutine %q exited with error:", getGoroutineName(ctx), err)
				}
			}
		}()

		return fn(ctx)
	})
}

// Wait for all goroutines in the group to finish, and return returns
// an error if any of the workers errored or timed out.
//
// Once the group has initiated hard shutdown (either a 2nd shutdown
// signal was received, or the parent context is <-Done()), Wait will
// return within the HardShutdownTimeout passed to NewGroup.  If a
// poorly-behaved goroutine is still running at the end of that time,
// it is left running, and an error is returned.
func (g *Group) Wait() error {
	// 1. Wait for the worker goroutines to finish (or time out)
	shutdownCompleted := make(chan error)
	go func() {
		shutdownCompleted <- g.inner.Wait()
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

	// 2. Belt-and-suspenders: Make sure that anything branched
	// from our Context observes that this group is no longer
	// running.
	g.hardCancel()

	// 3. Log the result and return
	if ret != nil && !g.cfg.DisableLogging {
		ctx := WithGoroutineName(g.baseCtx, ":shutdown_status")
		logGoroutineStatuses(ctx, "final goroutine statuses", dlog.Infof, g.List())
		if timedOut {
			logGoroutineTraces(ctx, "final goroutine stack traces", dlog.Errorf)
		}
	}
	return ret
}

// List wraps derrgroup.Group.List().
func (g *Group) List() map[string]derrgroup.GoroutineState {
	return g.inner.List()
}

type groupKey struct{}

// ParentGroup returns the Group that manages this goroutine/Context.
// If the Context is not managed by a Group, then nil is returned.
func ParentGroup(ctx context.Context) *Group {
	group := ctx.Value(groupKey{})
	if group == nil {
		return nil
	}
	return group.(*Group)
}
