package main

import (
	"github.com/datawire/ambassador-oauth/app"
	"github.com/datawire/ambassador-oauth/cmd/ambassador-oauth/config"
	"github.com/datawire/ambassador-oauth/cmd/ambassador-oauth/discovery"
	"github.com/datawire/ambassador-oauth/cmd/ambassador-oauth/logger"
	"github.com/datawire/ambassador-oauth/cmd/ambassador-oauth/secret"
	"github.com/datawire/ambassador-oauth/middleware"
	"github.com/urfave/negroni"
)

func main() {
	// Config
	config := config.New()
	if config == nil {
		panic("config object cannot be nil")
	}

	// Logger
	logger := logger.New(config)
	if logger == nil {
		panic("logger object cannot be nil")
	}

	// RSA keys
	secret := secret.New(config, logger)
	if secret == nil {
		logger.Fatal("keys object cannot be nil")
	}

	// Certificate Util
	discovery := discovery.New(config)
	if discovery == nil {
		logger.Fatal("certificate util object cannot be nil")
	}

	// K8s controller
	ctrl := &app.Controller{
		Logger: logger,
		Config: config,
	}
	ctrl.Watch()

	// Handler
	hdr := app.Handler{
		Config: config,
		Logger: logger,
		Ctrl:   ctrl,
		Secret: secret,
	}

	// Middlewares
	loggerMW := &middleware.Logger{Logger: logger}

	recoveryMW := &negroni.Recovery{
		Logger:     logger,
		PrintStack: false,
		StackAll:   false,
		StackSize:  1024 * 8,
		Formatter:  &negroni.TextPanicFormatter{},
	}

	jwtMW := &middleware.JWT{
		Logger:    logger,
		Config:    config,
		Discovery: discovery,
	}

	callbackMW := &middleware.Callback{
		Logger: logger,
		Config: config,
		Secret: secret,
	}

	// Negroni
	common := negroni.New()
	common.Use(loggerMW)
	common.Use(recoveryMW)
	common.Use(callbackMW)
	common.Use(jwtMW)
	common.UseHandlerFunc(hdr.Authorize)

	// Server
	common.Run(":8080")
}
