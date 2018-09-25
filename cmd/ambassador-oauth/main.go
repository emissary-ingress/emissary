package main

import (
	"net/http"

	"github.com/datawire/ambassador-oauth/cmd/ambassador-oauth/client"

	"github.com/datawire/ambassador-oauth/cmd/ambassador-oauth/app"
	"github.com/datawire/ambassador-oauth/cmd/ambassador-oauth/config"
	"github.com/datawire/ambassador-oauth/cmd/ambassador-oauth/controller"
	"github.com/datawire/ambassador-oauth/cmd/ambassador-oauth/discovery"
	"github.com/datawire/ambassador-oauth/cmd/ambassador-oauth/logger"
	"github.com/datawire/ambassador-oauth/cmd/ambassador-oauth/secret"
)

func main() {
	c := config.New()
	l := logger.New(c)
	s := secret.New(c, l)
	d := discovery.New(c)
	cl := client.NewRestClient(c.BaseURL)

	ct := &controller.Controller{
		Config: c,
		Logger: l,
	}
	ct.Watch()

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
