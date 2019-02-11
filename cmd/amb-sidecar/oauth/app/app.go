package app

import (
	"net/http"
	"net/url"

	"github.com/gorilla/mux"
	"github.com/urfave/negroni"

	crd "github.com/datawire/apro/apis/getambassador.io/v1beta1"
	"github.com/datawire/apro/cmd/amb-sidecar/oauth/app/client"
	"github.com/datawire/apro/cmd/amb-sidecar/oauth/app/discovery"
	"github.com/datawire/apro/cmd/amb-sidecar/oauth/app/handler"
	"github.com/datawire/apro/cmd/amb-sidecar/oauth/app/middleware"
	"github.com/datawire/apro/cmd/amb-sidecar/oauth/app/secret"
	"github.com/datawire/apro/cmd/amb-sidecar/oauth/controller"
	"github.com/datawire/apro/cmd/amb-sidecar/types"
)

// App is used to wire up all the cmd application components.
type App struct {
	Config     types.Config
	Logger     types.Logger
	Controller *controller.Controller

	secret    *secret.Secret
	discovery *discovery.Discovery
	rest      *client.Rest
}

// Handler returns an app handler that should be consumed by an HTTP server.
func (a *App) Handler() (http.Handler, error) {
	if a.Logger == nil {
		panic("logger object cannot be nil")
	}
	if a.Controller == nil {
		panic("controller object cannot be nil")
	}

	var err error
	a.secret, err = secret.New(a.Config, a.Logger) // RSA keys
	if err != nil {
		return nil, err
	}

	discovery, err := discovery.New(a.Config)
	if err != nil {
		return nil, err
	}

	a.Config.IssuerURL = discovery.Issuer

	a.discovery = discovery
	authorizationEndpointURL, err := url.Parse(a.discovery.AuthorizationEndpoint)
	if err != nil {
		return nil, err
	}

	tokenEndpointURL, err := url.Parse(a.discovery.TokenEndpoint)
	if err != nil {
		return nil, err
	}

	a.rest = client.NewRestClient(authorizationEndpointURL, tokenEndpointURL)

	// Handler
	auth := handler.Authorize{
		Config:    a.Config,
		Logger:    a.Logger.WithField("HANDLER", "authorize"),
		Ctrl:      a.Controller,
		Secret:    a.secret,
		Discovery: a.discovery,
	}

	cb := &handler.Callback{
		Logger: a.Logger.WithField("HANDLER", "callback"),
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

	n.Use(&middleware.Logger{Logger: a.Logger.WithField("MIDDLEWARE", "http")})

	n.Use(&negroni.Recovery{
		Logger:     a.Logger.WithField("MIDDLEWARE", "recovery"),
		PrintStack: false,
		StackAll:   false,
		StackSize:  1024 * 8,
		Formatter:  &negroni.TextPanicFormatter{},
	})

	n.Use(&middleware.CheckConfig{
		Config: a.Config,
	})

	n.Use(&middleware.DomainCheck{
		Logger: a.Logger.WithField("MIDDLEWARE", "app_check"),
		Ctrl:   a.Controller,
	})

	n.Use(&middleware.PolicyCheck{
		Logger: a.Logger.WithField("MIDDLEWARE", "policy_check"),
		Ctrl:   a.Controller,
		DefaultRule: &crd.Rule{
			Scope:  crd.DefaultScope,
			Public: false,
		},
	})

	n.Use(&middleware.JWTCheck{
		Logger:    a.Logger.WithField("MIDDLEWARE", "jwt_check"),
		Discovery: a.discovery,
		Config:    a.Config,
	})

	n.UseHandler(r)

	return n, nil
}
