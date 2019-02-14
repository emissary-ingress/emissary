package main

import (
	"context"
	"net/http"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"github.com/datawire/apro/cmd/amb-sidecar/oauth/app"
	"github.com/datawire/apro/cmd/amb-sidecar/oauth/controller"
	"github.com/datawire/apro/cmd/amb-sidecar/types"
)

func init() {
	cmd := &cobra.Command{
		Use:   "auth",
		Short: "Run OAuth service",
	}

	afterParse := types.InitializeFlags(cmd.Flags())

	cmd.RunE = func(*cobra.Command, []string) error {
		l := logrus.New()

		// Sets custom formatter.
		customFormatter := new(logrus.TextFormatter)
		customFormatter.TimestampFormat = "2006-01-02 15:04:05"
		l.SetFormatter(customFormatter)

		cfg := afterParse()
		if cfg.Error != nil {
			// This is a non-fatal error.  Even with an
			// invalid configuration, we continue to run,
			// but serve a 5XX error page.
			l.Errorf("config error: %v", cfg.Error)
		}

		customFormatter.FullTimestamp = true
		// Sets log level.
		if level, err := logrus.ParseLevel(cfg.LogLevel); err == nil {
			l.SetLevel(level)
		} else {
			l.Errorf("%v. Setting info log level as default.", err)
			l.SetLevel(logrus.InfoLevel)
		}

		group := NewGroup(context.Background(), cfg, func(name string) types.Logger {
			return types.WrapLogrus(l)
		})
		cmdAuth(group)
		return group.Wait()
	}

	argparser.AddCommand(cmd)
}

// cmdAuth runs the auth service.
//
//  - `softCtx` being canceled triggers a graceful shutdown
//  - `hardCtx` being canceled triggers a not-so-graceful shutdown
//  - register goroutines with `group`
func cmdAuth(group *Group) {
	// The gist here is that we have 2 main goroutines:
	// - the k8s controller, witch watches for CRD changes
	// - the HTTP server than handles auth requests
	// The HTTP server queries k8s controller for the current CRD
	// state.

	ct := &controller.Controller{}

	group.Go("auth_http", func(hardCtx, softCtx context.Context, cfg types.Config, l types.Logger) error {
		ct.Config = cfg
		ct.Logger = l.WithField("MAIN", "auth-k8s")
		ct.Watch(softCtx)
		return nil
	})

	group.Go("auth_http", func(hardCtx, softCtx context.Context, cfg types.Config, l types.Logger) error {
		httpHandler, err := app.NewHandler(cfg, l, ct)
		if err != nil {
			return err
		}
		server := &http.Server{
			Addr:     ":8080",
			Handler:  httpHandler,
			ErrorLog: l.WithField("MAIN", "auth-http").StdLogger(types.LogLevelError),
		}
		return listenAndServeWithContext(hardCtx, softCtx, server)
	})
}
