package app

import (
	"net/http"

	"github.com/pkg/errors"
	"github.com/urfave/negroni"

	crd "github.com/datawire/apro/apis/getambassador.io/v1beta1"
	"github.com/datawire/apro/cmd/amb-sidecar/oauth/app/client"
	"github.com/datawire/apro/cmd/amb-sidecar/oauth/app/discovery"
	"github.com/datawire/apro/cmd/amb-sidecar/oauth/app/handler"
	"github.com/datawire/apro/cmd/amb-sidecar/oauth/app/middleware"
	_secret "github.com/datawire/apro/cmd/amb-sidecar/oauth/app/secret"
	"github.com/datawire/apro/cmd/amb-sidecar/oauth/controller"
	"github.com/datawire/apro/cmd/amb-sidecar/types"
)

// Handler returns an app handler that should be consumed by an HTTP server.
func NewHandler(config types.Config, logger types.Logger, controller *controller.Controller) (http.Handler, error) {
	secret, err := _secret.New(config, logger) // RSA keys
	if err != nil {
		return nil, errors.Wrap(err, "secret")
	}

	disco, err := discovery.New(config, logger)
	if err != nil {
		return nil, errors.Wrap(err, "discovery")
	}

	rest := client.NewRestClient(disco.AuthorizationEndpoint, disco.TokenEndpoint)

	n := negroni.New()

	// Middleware (most-outer is listed first, most-inner is listed last)
	n.Use(&middleware.Logger{Logger: logger.WithField("MIDDLEWARE", "http")})
	n.Use(&negroni.Recovery{
		Logger:     logger.WithField("MIDDLEWARE", "recovery"),
		PrintStack: false,
		StackAll:   false,
		StackSize:  1024 * 8,
		Formatter:  &negroni.TextPanicFormatter{},
	})
	n.Use(&middleware.DomainCheck{
		Logger: logger.WithField("MIDDLEWARE", "app_check"),
		Ctrl:   controller,
	})
	n.Use(&middleware.PolicyCheck{
		Logger: logger.WithField("MIDDLEWARE", "policy_check"),
		Ctrl:   controller,
		DefaultRule: &crd.Rule{
			Scope:  crd.DefaultScope,
			Public: false,
		},
	})
	n.Use(&middleware.JWTCheck{
		Logger:    logger.WithField("MIDDLEWARE", "jwt_check"),
		Discovery: disco,
		Config:    config,
		IssuerURL: disco.Issuer,
	})

	// Final handler (most-inner of all)
	r := http.NewServeMux()
	r.Handle("/", &handler.Authorize{
		Config:    config,
		Logger:    logger.WithField("HANDLER", "authorize"),
		Ctrl:      controller,
		Secret:    secret,
		Discovery: disco,
	})
	r.Handle("/callback", &handler.Callback{
		Logger: logger.WithField("HANDLER", "callback"),
		Secret: secret,
		Ctrl:   controller,
		Rest:   rest,
	})
	n.UseHandler(r)

	return n, nil
}
