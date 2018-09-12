package main

import (
	"github.com/datawire/ambassador-oauth/app"
	"github.com/datawire/ambassador-oauth/cmd/ambassador-oauth/config"
	"github.com/datawire/ambassador-oauth/cmd/ambassador-oauth/logger"
	"github.com/datawire/ambassador-oauth/middleware"
	"github.com/urfave/negroni"
)

// PKey secret is used to sign the authorization state value.
const PKey = "vg=pgHoAAWgCsGuKBX,U3qrUGmqrPGE3" // TODO(gsagula): make PKey cli configurable.

func main() {
	// Config
	cfg := config.New()

	// Logger
	log := logger.New(cfg)

	// K8s controller
	ctrl := &app.Controller{Logger: log, Config: cfg}
	ctrl.Watch()

	// Handler
	hdr := app.Handler{Config: cfg, Logger: log, Ctrl: ctrl, PrivateKey: PKey}

	// Common
	common := negroni.New()
	common.Use(&middleware.Logger{Logger: log})

	// TODO(gsagula): make PrintStack cli configurable.
	common.Use(&negroni.Recovery{
		Logger:     log,
		PrintStack: true,
		StackAll:   false,
		StackSize:  1024 * 8,
		Formatter:  &negroni.TextPanicFormatter{},
	})
	common.Use(&middleware.Callback{Logger: log, Config: cfg, PrivateKey: PKey})

	// TODO(gsagula): replace this with our own middleware.
	jwt := &middleware.Jwt{Logger: log, Config: cfg}
	hdr.Jwt = jwt.Middleware()
	common.UseFunc(hdr.Jwt.HandlerWithNext)
	common.UseHandlerFunc(hdr.Authorize)

	// Server
	common.Run(":8080")
}
