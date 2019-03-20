package app

import (
	"net/http"
	"strings"

	"github.com/pkg/errors"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
	"google.golang.org/grpc"

	crd "github.com/datawire/apro/apis/getambassador.io/v1beta2"
	_secret "github.com/datawire/apro/cmd/amb-sidecar/filters/app/secret"
	"github.com/datawire/apro/cmd/amb-sidecar/filters/controller"
	"github.com/datawire/apro/cmd/amb-sidecar/types"
	"github.com/datawire/apro/lib/filterapi"
	"github.com/datawire/apro/lib/filterapi/filterutil"
)

// Handler returns an app handler that should be consumed by an HTTP server.
func NewFilterMux(config types.Config, logger types.Logger, controller *controller.Controller) (http.Handler, error) {
	secret, err := _secret.New(config, logger) // RSA keys
	if err != nil {
		return nil, errors.Wrap(err, "secret")
	}

	filterMux := &FilterMux{
		DefaultRule: &crd.Rule{
			Filters: nil,
		},
		Controller:   controller,
		OAuth2Secret: secret,
		Logger:       logger,
	}

	grpcServer := grpc.NewServer()
	filterapi.RegisterFilterService(grpcServer, filterutil.HandlerToFilter(filterMux))

	// The net/http.Server doesn't support h2c (unencrypted
	// HTTP/2) built-in.  Since we want to have gRPC and plain
	// HTTP/1 on the same unencrypted port, need h2c.
	// Fortunately, x/net has an h2c implementation we can use.
	return h2c.NewHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.ProtoMajor == 2 && strings.HasPrefix(r.Header.Get("Content-Type"), "application/grpc") {
			grpcServer.ServeHTTP(w, r)
		} else {
			filterMux.ServeHTTP(w, r)
		}
	}), &http2.Server{}), nil
}
