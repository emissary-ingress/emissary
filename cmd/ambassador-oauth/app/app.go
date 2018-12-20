package app

import (
	"net/http"

	"github.com/datawire/ambassador-oauth/cmd/ambassador-oauth/client"
	"github.com/gorilla/mux"

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
	Rest       *client.Rest
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

	// Handler
	auth := handler.Authorize{
		Config: a.Config,
		Logger: a.Logger.WithFields(logrus.Fields{"HANDLER": "authorize"}),
		Ctrl:   a.Controller,
		Secret: a.Secret,
	}

	cb := &handler.Callback{
		Logger: a.Logger.WithFields(logrus.Fields{"HANDLER": "callback"}),
		Config: a.Config,
		Secret: a.Secret,
		Ctrl:   a.Controller,
		Rest:   a.Rest,
	}

	// Router
	r := mux.NewRouter()

	r.HandleFunc("/callback", cb.Check)
	r.PathPrefix("/").HandlerFunc(auth.Check)

	// Middleware
	n := negroni.New()

	n.Use(&middleware.Logger{Logger: a.Logger.WithFields(logrus.Fields{"MIDDLEWARE": "http"})})

	n.Use(&negroni.Recovery{
		Logger:     a.Logger.WithFields(logrus.Fields{"MIDDLEWARE": "recovery"}),
		PrintStack: false,
		StackAll:   false,
		StackSize:  1024 * 8,
		Formatter:  &negroni.TextPanicFormatter{},
	})

	n.Use(&middleware.CheckConfig{
		Config: a.Config,
	})

	n.Use(&middleware.DomainCheck{
		Logger: a.Logger.WithFields(logrus.Fields{"MIDDLEWARE": "app_check"}),
		Config: a.Config,
		Ctrl:   a.Controller,
	})

	n.Use(&middleware.PolicyCheck{
		Logger: a.Logger.WithFields(logrus.Fields{"MIDDLEWARE": "policy_check"}),
		Ctrl:   a.Controller,
	})

	n.Use(&middleware.JWTCheck{
		Logger:    a.Logger.WithFields(logrus.Fields{"MIDDLEWARE": "jwt_check"}),
		Config:    a.Config,
		Discovery: a.Discovery,
	})

	n.UseHandler(r)

	return n
}
