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
//  - handles SIGINT and SIGTERM
//  - manages Context for you
//  - adds hard/soft cancelation
//  - does some minimal logging
type Group struct {
	hardCtx       context.Context
	softCtx       context.Context
	loggerFactory func(name string) dlog.Logger
	inner         *derrgroup.Group
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

// NewGroup returns a new Group.
func NewGroup(ctx context.Context, loggerFactory func(name string) dlog.Logger) *Group {
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	hardCtx, hardCancel := context.WithCancel(ctx)
	softCtx, softCancel := context.WithCancel(hardCtx)

	ret := &Group{
		hardCtx:       hardCtx,
		softCtx:       softCtx,
		loggerFactory: loggerFactory,
		inner:         derrgroup.NewGroup(softCancel),
	}

	ret.Go("supervisor", func(hardCtx, softCtx context.Context) error {
		<-softCtx.Done()
		dlog.Infoln(hardCtx, "shutting down...")
		return nil
	})

	ret.Go("signal_handler", func(hardCtx, softCtx context.Context) error {
		defer func() {
			// If we receive another signal after
			// graceful-shutdown, we should trigger a
			// not-so-graceful shutdown.
			go func() {
				sig := <-sigs
				dlog.Errorln(hardCtx, errors.Errorf("received signal %v", sig))
				logGoroutines(hardCtx, dlog.Errorf, ret.List())
				hardCancel()
				// keep logging signals
				for sig := range sigs {
					dlog.Errorln(hardCtx, errors.Errorf("received signal %v", sig))
					logGoroutines(hardCtx, dlog.Errorf, ret.List())
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

	return ret
}

// Go wraps derrgroup.Group.Go().
//
//  - `softCtx` being canceled should trigger a graceful shutdown
//  - `hardCtx` being canceled should trigger a not-so-graceful shutdown
func (g *Group) Go(name string, fn func(hardCtx, softCtx context.Context) error) {
	g.inner.Go(name, func() error {
		logger := g.loggerFactory(name)
		hardCtx := dlog.WithLogger(g.hardCtx, logger)
		softCtx := dlog.WithLogger(g.softCtx, logger)
		err := fn(hardCtx, softCtx)
		if err == nil {
			logger.Debugln("goroutine exited without error")
		} else {
			logger.Errorln("goroutine exited with error:", err)
		}
		return err
	})
}

// Wait wraps derrgroup.Group.Wait().
func (g *Group) Wait() error {
	ret := g.inner.Wait()
	if ret != nil {
		ctx := dlog.WithLogger(g.hardCtx, g.loggerFactory("shutdown_status"))
		logGoroutines(ctx, dlog.Infof, g.List())
	}
	return ret
}

// List wraps derrgroup.Group.List().
func (g *Group) List() map[string]derrgroup.GoroutineState {
	return g.inner.List()
}
