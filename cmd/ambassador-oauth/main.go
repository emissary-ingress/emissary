package main

import (
	"net/http"

	"github.com/datawire/apro/cmd/ambassador-oauth/client"
	"github.com/sirupsen/logrus"

	"github.com/datawire/apro/cmd/ambassador-oauth/app"
	"github.com/datawire/apro/cmd/ambassador-oauth/config"
	"github.com/datawire/apro/cmd/ambassador-oauth/controller"
	"github.com/datawire/apro/cmd/ambassador-oauth/discovery"
	"github.com/datawire/apro/cmd/ambassador-oauth/logger"
	"github.com/datawire/apro/cmd/ambassador-oauth/secret"
)

func main() {
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
