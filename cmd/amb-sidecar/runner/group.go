package runner

import (
	"context"
	"os"
	"os/signal"
	"sort"
	"syscall"

	"github.com/datawire/ambassador/pkg/dlog"
	"github.com/pkg/errors"

	"github.com/datawire/apro/cmd/amb-sidecar/types"
)

// Group is a wrapper around golang.org/x/sync/errgroup.Group (err, a
// fork of errgroup) that:
//  - includes application-specific arguments to your worker functions
//  - handles SIGINT and SIGTERM
type Group struct {
	hardCtx       context.Context
	softCtx       context.Context
	cfg           types.Config
	loggerFactory func(name string) dlog.Logger
	inner         *llGroup
}

func logGoroutines(l dlog.Logger, list map[string]GoroutineState) {
	l.Errorln("  goroutine shutdown status:")
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
		l.Errorf("    %-*s: %s", nameWidth, name, list[name])
	}
}

// NewGroup returns a new Group.
func NewGroup(ctx context.Context, cfg types.Config, loggerFactory func(name string) dlog.Logger) *Group {
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	hardCtx, hardCancel := context.WithCancel(ctx)
	softCtx, softCancel := context.WithCancel(hardCtx)

	ret := &Group{
		hardCtx:       hardCtx,
		softCtx:       softCtx,
		cfg:           cfg,
		loggerFactory: loggerFactory,
		inner:         newLLGroup(softCancel),
	}

	ret.Go("signal_handler", func(_, _ context.Context, _ types.Config, l dlog.Logger) error {
		defer func() {
			// If we receive another signal after
			// graceful-shutdown, we should trigger a
			// not-so-graceful shutdown.
			go func() {
				sig := <-sigs
				l.Errorln(errors.Errorf("received signal %v", sig))
				logGoroutines(l, ret.List())
				hardCancel()
				// keep logging signals
				for sig := range sigs {
					l.Errorln(errors.Errorf("received signal %v", sig))
					logGoroutines(l, ret.List())
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

// Go wraps llGroup.Go().
//
//  - `softCtx` being canceled should trigger a graceful shutdown
//  - `hardCtx` being canceled should trigger a not-so-graceful shutdown
func (g *Group) Go(name string, fn func(hardCtx, softCtx context.Context, cfg types.Config, logger dlog.Logger) error) {
	g.inner.Go(name, func() error {
		return fn(g.hardCtx, g.softCtx, g.cfg, g.loggerFactory(name))
	})
}

// Wait wraps llGroup.Wait().
func (g *Group) Wait() error {
	return g.inner.Wait()
}

// List wraps llGroup.List().
func (g *Group) List() map[string]GoroutineState {
	return g.inner.List()
}
