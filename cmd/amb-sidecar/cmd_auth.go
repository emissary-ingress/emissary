package main

import (
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

func init() {
	argparser.AddCommand(&cobra.Command{
		Use:   "auth",
		Short: "Run OAuth service",
		Run:   cmdAuth,
	})
}

func cmdAuth(flags *cobra.Command, args []string) {
	c := config.New()
	l := logger.New(c)
	s := secret.New(c, l)
	d := discovery.New(c)
	cl := client.NewRestClient(c.BaseURL)

	ct := &controller.Controller{
		Config: c,
		Logger: l.WithFields(logrus.Fields{"MAIN": "controller"}),
	}

	go ct.Watch()

	a := app.App{
		Config:     c,
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
