package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"golang.org/x/sync/errgroup"

	"github.com/datawire/apro/cmd/amb-sidecar/oauth/app"
	"github.com/datawire/apro/cmd/amb-sidecar/oauth/controller"
	"github.com/datawire/apro/cmd/amb-sidecar/types"
)

func init() {
	var cfg *types.Config

	cmd := &cobra.Command{
		Use:   "auth",
		Short: "Run OAuth service",
		RunE: func(*cobra.Command, []string) error {
			l := logrus.New()

			// Sets custom formatter.
			customFormatter := new(logrus.TextFormatter)
			customFormatter.TimestampFormat = "2006-01-02 15:04:05"
			l.SetFormatter(customFormatter)

			customFormatter.FullTimestamp = true
			// Sets log level.
			if level, err := logrus.ParseLevel(cfg.LogLevel); err == nil {
				l.SetLevel(level)
			} else {
				l.Errorf("%v. Setting info log level as default.", err)
				l.SetLevel(logrus.InfoLevel)
			}

			// Initialize hardCtx/softCtx/group to manage
			// our goroutines and graceful shutdown.
			sigs := make(chan os.Signal, 1)
			signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
			hardCtx, hardCancel := context.WithCancel(context.Background())
			group, softCtx := errgroup.WithContext(hardCtx)
			group.Go(func() error {
				defer func() {
					// If we recieve another signal after
					// gracefull-shutdown, we should trigger a
					// not-so-graceful shutdown.
					go func() {
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

			return cmdAuth(hardCtx, softCtx, group, cfg, l)
		},
	}

	afterParse := types.InitializeFlags(cmd.Flags())

	cmd.PreRun = func(cmd *cobra.Command, args []string) {
		cfg = afterParse()
		if cfg.Error != nil {
			// This is a non-fatal error.  Even with an
			// invalid configuration, we continue to run,
			// but serve a 5XX error page.
			log.Printf("config error: %v", cfg.Error)
		}
	}

	argparser.AddCommand(cmd)
}

// cmdAuth runs the auth service.
//
//  - `softCtx` being canceled triggers a graceful shutdown
//  - `hardCtx` being canceled triggers a not-so-graceful shutdown
//  - register goroutines with `group`
func cmdAuth(
	hardCtx, softCtx context.Context, group *errgroup.Group, // for keeping track of goroutines
	authCfg *types.Config, // config, tells us what to do
	l *logrus.Logger, // where to log to
) error {
	// The gist here is that we have 2 main goroutines:
	// - the k8s controller, witch watches for CRD changes
	// - the HTTP server than handles auth requests
	// The HTTP server queries k8s controller for the current CRD
	// state.

	ct := &controller.Controller{
		Config: authCfg,
		Logger: l.WithFields(logrus.Fields{"MAIN": "controller"}),
	}

	group.Go(func() error {
		ct.Watch(softCtx)
		return nil
	})

	group.Go(func() error {
		a := app.App{
			Config:     authCfg,
			Logger:     l,
			Controller: ct,
		}
		server := &http.Server{
			Addr:     ":8080",
			Handler:  a.Handler(),
			ErrorLog: log.New(l.WriterLevel(logrus.ErrorLevel), "auth-http", 0),
		}
		return listenAndServeWithContext(hardCtx, softCtx, server)
	})

	return group.Wait()
}

// listenAndServeWithContext runs server.ListenAndServer() on an
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
