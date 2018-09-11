package main

import (
	"log"

	"github.com/datawire/ambassador-oauth/app"
	"github.com/datawire/ambassador-oauth/cmd/ambassador-oauth/config"
	"github.com/datawire/ambassador-oauth/cmd/ambassador-oauth/logger"
	"github.com/datawire/ambassador-oauth/middleware"
	"github.com/urfave/negroni"
)

// StateSignature secret is used to sign the authorization state value.
const PKey = "vg=pgHoAAWgCsGuKBX,U3qrUGmqrPGE3"

func main() {
	// Config
	c, err := config.NewConfig()
	if err != nil {
		log.Fatalf("terminating: %v", err)
	}

	// Logger
	l := logger.NewLogger(c)

	// K8s controller
	ctrl := &app.Controller{Logger: l, Config: c}
	ctrl.Watch()

	// Handler
	h := app.Handler{Config: c, Logger: l, Ctrl: ctrl, PrivateKey: PKey}

	// Common
	// TODO(gsagula): get rid of the old POC design and refactor jwt midleware & authorize handler.
	common := negroni.New()
	common.Use(&middleware.Logger{Logger: l})
	common.Use(negroni.NewRecovery())
	common.Use(&middleware.Callback{Logger: l, Config: c, PrivateKey: PKey})
	jwt := &middleware.Jwt{Logger: l, Config: c}
	h.Jwt = jwt.Middleware()
	common.UseFunc(h.Jwt.HandlerWithNext)
	common.UseHandlerFunc(h.Authorize)

	// Server
	common.Run(":8080")
}
