package main

import (
	"context"
	"log"
	"net/http"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"github.com/datawire/apro/cmd/amb-sidecar/oauth/app"
	"github.com/datawire/apro/cmd/amb-sidecar/oauth/client"
	"github.com/datawire/apro/cmd/amb-sidecar/oauth/config"
	"github.com/datawire/apro/cmd/amb-sidecar/oauth/controller"
	"github.com/datawire/apro/cmd/amb-sidecar/oauth/discovery"
	"github.com/datawire/apro/cmd/amb-sidecar/oauth/logger"
	"github.com/datawire/apro/cmd/amb-sidecar/oauth/secret"
)

var authCfg *config.Config

func init() {
	cmd := &cobra.Command{
		Use:   "auth",
		Short: "Run OAuth service",
		Run:   cmdAuth,
	}

	afterParse := config.InitializeFlags(cmd.Flags())

	cmd.PreRun = func(cmd *cobra.Command, args []string) {
		authCfg = afterParse()
		if authCfg.Error != nil {
			// This is a non-fatal error.  Even with an
			// invalid configuration, we continue to run,
			// but serve a 5XX error page.
			log.Printf("config error: %v", authCfg.Error)
		}
	}

	argparser.AddCommand(cmd)
}

func cmdAuth(flags *cobra.Command, args []string) {
	l := logger.New(authCfg)
	s := secret.New(authCfg, l)
	d := discovery.New(authCfg)
	cl := client.NewRestClient(authCfg.BaseURL)

	ct := &controller.Controller{
		Config: authCfg,
		Logger: l.WithFields(logrus.Fields{"MAIN": "controller"}),
	}

	go func() {
		err := ct.Watch(context.Background())
		if err != nil {
			l.Fatal(err)
		}
	}()

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
		l.Fatal(err)
	}
}
