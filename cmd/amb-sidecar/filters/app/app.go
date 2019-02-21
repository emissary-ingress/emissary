package app

import (
	"net/http"

	"github.com/pkg/errors"
	"github.com/urfave/negroni"

	crd "github.com/datawire/apro/apis/getambassador.io/v1beta2"
	"github.com/datawire/apro/cmd/amb-sidecar/filters/app/middleware"
	_secret "github.com/datawire/apro/cmd/amb-sidecar/filters/app/secret"
	"github.com/datawire/apro/cmd/amb-sidecar/filters/controller"
	"github.com/datawire/apro/cmd/amb-sidecar/types"
)

// Handler returns an app handler that should be consumed by an HTTP server.
func NewHandler(config types.Config, logger types.Logger, controller *controller.Controller) (http.Handler, error) {
	secret, err := _secret.New(config, logger) // RSA keys
	if err != nil {
		return nil, errors.Wrap(err, "secret")
	}

	n := negroni.New()

	// Middleware (most-outer is listed first, most-inner is listed last)
	n.Use(&middleware.Logger{Logger: logger})
	// Final handler (most-inner of all)
	n.UseHandler(&FilterHandler{
		DefaultRule: &crd.Rule{
			Public: false,
		},
		Controller:   controller,
		OAuth2Secret: secret,
	})

	return n, nil
}
