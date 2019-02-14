package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/pkg/errors"
	"golang.org/x/sync/errgroup"

	"github.com/datawire/apro/cmd/amb-sidecar/types"
)

// Group is a wrapper around golang.org/x/sync/errgroup.Group that
//  - includes application-specific arguments to your worker functions
//  - handles SIGINT and SIGTERM
type Group struct {
	hardCtx       context.Context
	softCtx       context.Context
	cfg           types.Config
	loggerFactory func(name string) types.Logger
	inner         *errgroup.Group
}

// NewGroup returns a new Group.
func NewGroup(ctx context.Context, cfg types.Config, loggerFactory func(name string) types.Logger) *Group {
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	hardCtx, hardCancel := context.WithCancel(ctx)

	group, softCtx := errgroup.WithContext(hardCtx)
	group.Go(func() error {
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
		inner:         group,
	}
}

// Go wraps errgroup.Group.Go().
func (g *Group) Go(name string, fn func(hardCtx, softCtx context.Context, cfg types.Config, logger types.Logger) error) {
	g.inner.Go(func() error {
		return fn(g.hardCtx, g.softCtx, g.cfg, g.loggerFactory(name))
	})
}

// Go wraps errgroup.Group.Wait().
func (g *Group) Wait() error {
	return g.inner.Wait()
}

// listenAndServeWithContext runs server.ListenAndServe() on an
// http.Server(), but properly calls server.Shutdown when the context
// is canceled.
//
// softCtx should be a child context of hardCtx.  softCtx being
// canceled triggers server.Shutdown().  If hardCtx being cacneled
// triggers that .Shutdown() to kill any live requests and return,
// instead of waiting for them to be completed gracefully.
func listenAndServeWithContext(hardCtx, softCtx context.Context, server *http.Server) error {
	serverCh := make(chan error)
	go func() {
		serverCh <- server.ListenAndServe()
	}()
	select {
	case err := <-serverCh:
		return err
	case <-softCtx.Done():
		return server.Shutdown(hardCtx)
	}
}
