package main

import (
	"context"
	"log"
	"net/http"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"golang.org/x/sync/errgroup"

	"github.com/datawire/apro/cmd/amb-sidecar/config"
	"github.com/datawire/apro/cmd/amb-sidecar/oauth/app"
	"github.com/datawire/apro/cmd/amb-sidecar/oauth/client"
	"github.com/datawire/apro/cmd/amb-sidecar/oauth/controller"
	"github.com/datawire/apro/cmd/amb-sidecar/oauth/discovery"
	"github.com/datawire/apro/cmd/amb-sidecar/oauth/secret"
)

func init() {
	var cfg *config.Config

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

			return cmdAuth(cfg, l)
		},
	}

	afterParse := config.InitializeFlags(cmd.Flags())

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

func cmdAuth(authCfg *config.Config, l *logrus.Logger) error {
	s := secret.New(authCfg, l)
	d := discovery.New(authCfg)
	cl := client.NewRestClient(authCfg.BaseURL)

	ct := &controller.Controller{
		Config: authCfg,
		Logger: l.WithFields(logrus.Fields{"MAIN": "controller"}),
	}

	group, _ := errgroup.WithContext(context.Background())

	group.Go(func() error {
		ct.Watch()
		return nil
	})

	group.Go(func() error {
		a := app.App{
			Config:     authCfg,
			Logger:     l,
			Secret:     s,
			Discovery:  d,
			Controller: ct,
			Rest:       cl,
		}
		return http.ListenAndServe(":8080", a.Handler())
	})

	return group.Wait()
}
