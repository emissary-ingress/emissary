package runner

import (
	"context"
	"os"
	"os/signal"
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

// NewGroup returns a new Group.
func NewGroup(ctx context.Context, cfg types.Config, loggerFactory func(name string) dlog.Logger) *Group {
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	hardCtx, hardCancel := context.WithCancel(ctx)

	inner, softCtx := WithContext(hardCtx)
	inner.Go(func() error {
		defer func() {
			// If we receive another signal after
			// graceful-shutdown, we should trigger a
			// not-so-graceful shutdown.
			go func() {
				l := loggerFactory("signal_handler")
				sig := <-sigs
				l.Errorln(errors.Errorf("received signal %v", sig))
				hardCancel()
				// keep logging signals
				for sig := range sigs {
					l.Errorln(errors.Errorf("received signal %v", sig))
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

	return &Group{
		hardCtx:       hardCtx,
		softCtx:       softCtx,
		cfg:           cfg,
		loggerFactory: loggerFactory,
		inner:         inner,
	}
}

// Go wraps llGroup.Go().
//
//  - `softCtx` being canceled should trigger a graceful shutdown
//  - `hardCtx` being canceled should trigger a not-so-graceful shutdown
func (g *Group) Go(name string, fn func(hardCtx, softCtx context.Context, cfg types.Config, logger dlog.Logger) error) {
	g.inner.Go(func() error {
		return fn(g.hardCtx, g.softCtx, g.cfg, g.loggerFactory(name))
	})
}

// Go wraps llGroup.Wait().
func (g *Group) Wait() error {
	return g.inner.Wait()
}
