package dgroup

import (
	"context"
	"os"
	"os/signal"
	"sort"
	"syscall"

	"github.com/pkg/errors"

	"github.com/datawire/ambassador/pkg/derrgroup"
	"github.com/datawire/ambassador/pkg/dlog"
)

// Group is a wrapper around
// github.com/datawire/ambassador/pkg/derrgroup.Group that:
//  - (optionally) handles SIGINT and SIGTERM
//  - manages Context for you
//  - adds hard/soft cancelation
//  - (optionally) does some minimal logging
type Group struct {
	cfg     GroupConfig
	hardCtx context.Context
	softCtx context.Context
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
	EnableSignalHandling bool
	WorkerLogger         func(name string) dlog.Logger
}

// NewGroup returns a new Group.
func NewGroup(ctx context.Context, cfg GroupConfig) *Group {
	hardCtx, hardCancel := context.WithCancel(ctx)
	softCtx, softCancel := context.WithCancel(hardCtx)

	g := &Group{
		cfg:     cfg,
		hardCtx: hardCtx,
		softCtx: softCtx,
		inner:   derrgroup.NewGroup(softCancel),
	}

	if g.cfg.WorkerLogger != nil {
		g.Go("supervisor", func(hardCtx, softCtx context.Context) error {
			<-softCtx.Done()
			dlog.Infoln(hardCtx, "shutting down...")
			return nil
		})
	}

	if g.cfg.EnableSignalHandling {
		g.Go("signal_handler", func(hardCtx, softCtx context.Context) error {
			sigs := make(chan os.Signal, 1)
			signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

			defer func() {
				// If we receive another signal after
				// graceful-shutdown, we should trigger a
				// not-so-graceful shutdown.
				go func() {
					sig := <-sigs
					if g.cfg.WorkerLogger != nil {
						dlog.Errorln(hardCtx, errors.Errorf("received signal %v", sig))
						logGoroutines(hardCtx, dlog.Errorf, g.List())
					}
					hardCancel()
					// keep logging signals and draining 'sigs'--don't let 'sigs' block
					for sig := range sigs {
						if g.cfg.WorkerLogger != nil {
							dlog.Errorln(hardCtx, errors.Errorf("received signal %v", sig))
							logGoroutines(hardCtx, dlog.Errorf, g.List())
						}
					}
				}()
			}()

			select {
			case sig := <-sigs:
				return errors.Errorf("received signal %v", sig)
			case <-softCtx.Done():
				return nil
			}
		})
	}

	return g
}

// Go wraps derrgroup.Group.Go().
//
//  - `softCtx` being canceled should trigger a graceful shutdown
//  - `hardCtx` being canceled should trigger a not-so-graceful shutdown
func (g *Group) Go(name string, fn func(hardCtx, softCtx context.Context) error) {
	g.inner.Go(name, func() error {
		hardCtx, softCtx := g.hardCtx, g.softCtx
		var logger dlog.Logger
		if g.cfg.WorkerLogger != nil {
			logger = g.cfg.WorkerLogger(name)
			hardCtx = dlog.WithLogger(hardCtx, logger)
			softCtx = dlog.WithLogger(softCtx, logger)
		}
		err := fn(hardCtx, softCtx)
		if g.cfg.WorkerLogger != nil {
			if err == nil {
				logger.Debugln("goroutine exited without error")
			} else {
				logger.Errorln("goroutine exited with error:", err)
			}
		}
		return err
	})
}

// Wait wraps derrgroup.Group.Wait().
func (g *Group) Wait() error {
	ret := g.inner.Wait()
	if ret != nil && g.cfg.WorkerLogger != nil {
		ctx := dlog.WithLogger(g.hardCtx, g.cfg.WorkerLogger("shutdown_status"))
		logGoroutines(ctx, dlog.Infof, g.List())
	}
	return ret
}

// List wraps derrgroup.Group.List().
func (g *Group) List() map[string]derrgroup.GoroutineState {
	return g.inner.List()
}
