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

	cmd.RunE = func(*cobra.Command, []string) error {
		l := logrus.New()
		l.SetFormatter(&logrus.TextFormatter{
			TimestampFormat: "2006-01-02 15:04:05",
			FullTimestamp:   true,
		})

		cfg, warn, fatal := types.ConfigFromEnv()
		for _, err := range warn {
			l.Warnln("config error:", err)
		}
		for _, err := range fatal {
			l.Errorln("config error:", err)
		}
		if len(fatal) > 0 {
			return fatal[len(fatal)-1]
		}

		// cfg.LogLevel has already been validated in
		// ConfigFromEnv(), no need to error-check.
		level, _ := logrus.ParseLevel(cfg.LogLevel)
		l.SetLevel(level)

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
