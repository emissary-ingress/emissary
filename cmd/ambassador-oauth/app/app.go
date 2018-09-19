package app

import (
	"net/http"

	"github.com/datawire/ambassador-oauth/cmd/ambassador-oauth/config"
	"github.com/datawire/ambassador-oauth/cmd/ambassador-oauth/controller"
	"github.com/datawire/ambassador-oauth/cmd/ambassador-oauth/discovery"
	"github.com/datawire/ambassador-oauth/cmd/ambassador-oauth/secret"
	"github.com/datawire/ambassador-oauth/handler"
	"github.com/datawire/ambassador-oauth/middleware"
	"github.com/sirupsen/logrus"
	"github.com/urfave/negroni"
)

// App is used to wire up all the cmd application components.
type App struct {
	Config     *config.Config
	Logger     *logrus.Logger
	Secret     *secret.Secret
	Discovery  *discovery.Discovery
	Controller *controller.Controller
}

// Handler returns an app handler that should be consumed by an HTTP server.
func (a *App) Handler() http.Handler {
	// Config
	if a.Config == nil {
		panic("config object cannot be nil")
	}

	// Logger
	if a.Logger == nil {
		panic("logger object cannot be nil")
	}

	// RSA keys
	if a.Secret == nil {
		a.Logger.Fatal("keys object cannot be nil")
	}

	// Discovery
	if a.Discovery == nil {
		a.Logger.Fatal("certificate util object cannot be nil")
	}

	// K8s controller
	ctrl := &controller.Controller{
		Logger: a.Logger,
		Config: a.Config,
	}
	ctrl.Watch()

	// Handlers
	authz := handler.Authorize{
		Config: a.Config,
		Logger: a.Logger,
		Ctrl:   ctrl,
		Secret: a.Secret,
	}

	// Middlewares
	loggerMW := &middleware.Logger{Logger: a.Logger}

	recoveryMW := &negroni.Recovery{
		Logger:     a.Logger,
		PrintStack: false,
		StackAll:   false,
		StackSize:  1024 * 8,
		Formatter:  &negroni.TextPanicFormatter{},
	}

	jwtMW := &middleware.JWT{
		Logger:    a.Logger,
		Config:    a.Config,
		Discovery: a.Discovery,
	}

	callbackMW := &middleware.Callback{
		Logger: a.Logger,
		Config: a.Config,
		Secret: a.Secret,
	}

	// HTTP handler.
	// Note that middleware are executed in order.
	n := negroni.New()
	n.Use(loggerMW)
	n.Use(recoveryMW)
	n.Use(callbackMW)
	n.Use(jwtMW)
	n.UseHandlerFunc(authz.Check)

	return n
}
