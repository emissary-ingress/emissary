package app

import (
	"net/http"

	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
	"github.com/urfave/negroni"

	crd "github.com/datawire/apro/apis/getambassador.io/v1beta1"
	"github.com/datawire/apro/cmd/amb-sidecar/config"
	"github.com/datawire/apro/cmd/amb-sidecar/oauth/app/client"
	"github.com/datawire/apro/cmd/amb-sidecar/oauth/app/discovery"
	"github.com/datawire/apro/cmd/amb-sidecar/oauth/app/handler"
	"github.com/datawire/apro/cmd/amb-sidecar/oauth/app/middleware"
	"github.com/datawire/apro/cmd/amb-sidecar/oauth/app/secret"
	"github.com/datawire/apro/cmd/amb-sidecar/oauth/controller"
)

// App is used to wire up all the cmd application components.
type App struct {
	Config     *config.Config
	Logger     *logrus.Logger
	Controller *controller.Controller

	secret    *secret.Secret
	discovery *discovery.Discovery
	rest      *client.Rest
}

// Handler returns an app handler that should be consumed by an HTTP server.
func (a *App) Handler() http.Handler {
	if a.Config == nil {
		panic("config object cannot be nil")
	}
	if a.Logger == nil {
		panic("logger object cannot be nil")
	}
	if a.Controller == nil {
		panic("controller object cannot be nil")
	}

	a.secret = secret.New(a.Config, a.Logger) // RSA keys
	a.discovery = discovery.New(a.Config)
	a.rest = client.NewRestClient(a.Config.BaseURL)

	// Handler
	auth := handler.Authorize{
		Config: a.Config,
		Logger: a.Logger.WithFields(logrus.Fields{"HANDLER": "authorize"}),
		Ctrl:   a.Controller,
		Secret: a.secret,
	}

	cb := &handler.Callback{
		Logger: a.Logger.WithFields(logrus.Fields{"HANDLER": "callback"}),
		Secret: a.secret,
		Ctrl:   a.Controller,
		Rest:   a.rest,
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
		Ctrl:   a.Controller,
	})

	n.Use(&middleware.PolicyCheck{
		Logger: a.Logger.WithFields(logrus.Fields{"MIDDLEWARE": "policy_check"}),
		Ctrl:   a.Controller,
		DefaultRule: &crd.Rule{
			Scope:  crd.DefaultScope,
			Public: false,
		},
	})

	n.Use(&middleware.JWTCheck{
		Logger:    a.Logger.WithFields(logrus.Fields{"MIDDLEWARE": "jwt_check"}),
		Discovery: a.discovery,
		Config:    a.Config,
	})

	n.UseHandler(r)

	return n
}
