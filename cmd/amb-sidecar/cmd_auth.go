package main

import (
	"log"
	"net/http"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"github.com/datawire/apro/cmd/amb-sidecar/config"
	"github.com/datawire/apro/cmd/amb-sidecar/oauth/app"
	"github.com/datawire/apro/cmd/amb-sidecar/oauth/client"
	"github.com/datawire/apro/cmd/amb-sidecar/oauth/controller"
	"github.com/datawire/apro/cmd/amb-sidecar/oauth/discovery"
	"github.com/datawire/apro/cmd/amb-sidecar/oauth/logger"
	"github.com/datawire/apro/cmd/amb-sidecar/oauth/secret"
)

func init() {
	var cfg *config.Config

	cmd := &cobra.Command{
		Use:   "auth",
		Short: "Run OAuth service",
		RunE: func(*cobra.Command, []string) error {
			return cmdAuth(cfg, logger.New(cfg))
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

	go ct.Watch()

	a := app.App{
		Config:     authCfg,
		Logger:     l,
		Secret:     s,
		Discovery:  d,
		Controller: ct,
		Rest:       cl,
	}

	// Server
	if err := http.ListenAndServe(":8080", a.Handler()); err != nil {
		return err
	}
	return nil
}
