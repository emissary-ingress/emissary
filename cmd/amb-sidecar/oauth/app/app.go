package app

import (
	"net/http"

	"github.com/pkg/errors"
	"github.com/urfave/negroni"

	crd "github.com/datawire/apro/apis/getambassador.io/v1beta1"
	"github.com/datawire/apro/cmd/amb-sidecar/oauth/app/client"
	"github.com/datawire/apro/cmd/amb-sidecar/oauth/app/discovery"
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
	// Final handler (most-inner of all)
	n.UseHandler(&OAuth2Handler{
		DefaultRule: &crd.Rule{
			Scope:  crd.DefaultScope,
			Public: false,
		},
		Config:    config,
		Ctrl:      controller,
		Discovery: disco,
		IssuerURL: disco.Issuer,
		Logger:    logger.WithField("HANDLER", "oauth2"),
		Rest:      rest,
		Secret:    secret,
	})

	return n, nil
}
