package dgroup

import (
	"context"
	"os"
	"os/signal"
	"sort"
	"syscall"

	"github.com/pkg/errors"

	"github.com/datawire/ambassador/pkg/dcontext"
	"github.com/datawire/ambassador/pkg/derrgroup"
	"github.com/datawire/ambassador/pkg/dlog"
)

// Group is a wrapper around
// github.com/datawire/ambassador/pkg/derrgroup.Group that:
//  - (optionally) handles SIGINT and SIGTERM
//  - (configurable) manages Context for you
//  - (optionally) adds hard/soft cancellation
//  - (optionally) does some minimal logging
type Group struct {
	cfg     GroupConfig
	baseCtx context.Context
	inner   *derrgroup.Group
}

func logGoroutines(ctx context.Context, printf func(ctx context.Context, format string, args ...interface{}), list map[string]derrgroup.GoroutineState) {
	printf(ctx, "  goroutine shutdown status:")
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

// GroupConfig is a readable way of setting the configuration options
// for NewGroup.
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

	DisableLogging bool

	WorkerContext func(ctx context.Context, name string) context.Context
}

// NewGroup returns a new Group.
func NewGroup(ctx context.Context, cfg GroupConfig) *Group {
	cfg.EnableWithSoftness = cfg.EnableWithSoftness || cfg.EnableSignalHandling

	ctx, hardCancel := context.WithCancel(ctx)
	var softCancel context.CancelFunc
	if cfg.EnableWithSoftness {
		ctx = dcontext.WithSoftness(ctx)
		ctx, softCancel = context.WithCancel(ctx)
	} else {
		softCancel = hardCancel
	}

	g := &Group{
		cfg:     cfg,
		baseCtx: ctx,
		inner:   derrgroup.NewGroup(softCancel),
	}

	if !g.cfg.DisableLogging {
		g.Go("supervisor", func(ctx context.Context) error {
			<-ctx.Done()
			dlog.Infoln(ctx, "shutting down...")
			return nil
		})
	}

	if g.cfg.EnableSignalHandling {
		g.Go("signal_handler", func(ctx context.Context) error {
			sigs := make(chan os.Signal, 1)
			signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

			defer func() {
				// If we receive another signal after
				// graceful-shutdown, we should trigger a
				// not-so-graceful shutdown.
				go func() {
					sig := <-sigs
					if !g.cfg.DisableLogging {
						dlog.Errorln(ctx, errors.Errorf("received signal %v", sig))
						logGoroutines(ctx, dlog.Errorf, g.List())
					}
					hardCancel()
					// keep logging signals and draining 'sigs'--don't let 'sigs' block
					for sig := range sigs {
						if !g.cfg.DisableLogging {
							dlog.Errorln(ctx, errors.Errorf("received signal %v", sig))
							logGoroutines(ctx, dlog.Errorf, g.List())
						}
					}
				}()
			}()

			select {
			case sig := <-sigs:
				return errors.Errorf("received signal %v", sig)
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
func (g *Group) Go(name string, fn func(ctx context.Context) error) {
	g.inner.Go(name, func() error {
		ctx := g.baseCtx
		if g.cfg.WorkerContext != nil {
			ctx = g.cfg.WorkerContext(ctx, name)
		}
		err := fn(ctx)
		if !g.cfg.DisableLogging {
			if err == nil {
				dlog.Debugln(ctx, "goroutine exited without error")
			} else {
				dlog.Errorln(ctx, "goroutine exited with error:", err)
			}
		}
		return err
	})
}

// Wait wraps derrgroup.Group.Wait().
func (g *Group) Wait() error {
	ret := g.inner.Wait()
	if ret != nil && !g.cfg.DisableLogging {
		ctx := g.baseCtx
		if g.cfg.WorkerContext != nil {
			ctx = g.cfg.WorkerContext(ctx, "shutdown_status")
		}
		logGoroutines(ctx, dlog.Infof, g.List())
	}
	return ret
}

// List wraps derrgroup.Group.List().
func (g *Group) List() map[string]derrgroup.GoroutineState {
	return g.inner.List()
}
