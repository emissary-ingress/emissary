package app

import (
	"net/http"
	"strings"

	"github.com/pkg/errors"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
	"google.golang.org/grpc"

	k8sClientCoreV1 "k8s.io/client-go/kubernetes/typed/core/v1"

	crd "github.com/datawire/apro/apis/getambassador.io/v1beta2"
	"github.com/datawire/apro/cmd/amb-sidecar/filters/app/health"
	secret "github.com/datawire/apro/cmd/amb-sidecar/filters/app/secret"
	"github.com/datawire/apro/cmd/amb-sidecar/filters/controller"
	"github.com/datawire/apro/cmd/amb-sidecar/types"
	"github.com/datawire/apro/lib/filterapi"
)

// Handler returns an app handler that should be consumed by an HTTP server.
func NewFilterMux(
	config types.Config,
	logger types.Logger,
	controller *controller.Controller,
	secretsGetter k8sClientCoreV1.SecretsGetter,
) (http.Handler, error) {
	privKey, pubKey, err := secret.GetKeyPair(config, secretsGetter)
	if err != nil {
		return nil, errors.Wrap(err, "secret")
	}

	filterMux := &FilterMux{
		DefaultRule: &crd.Rule{
			Filters: nil,
		},
		Controller: controller,
		PrivateKey: privKey,
		PublicKey:  pubKey,
		Logger:     logger,
	}

	grpcServer := grpc.NewServer()
	filterapi.RegisterFilterService(grpcServer, filterMux)

	// register more health probes off the MultiProbe as
	probe := health.MultiProbe{
		Logger: logger,
	}

	// this is a probe that always returns true... it is admittedly "dumb", but if the HTTP server stops serving
	// this will fail and it forms the basis of the Probe API which we can use for subsequent more involved probes.
	probe.RegisterProbe("basic", &health.StaticProbe{Value: true})

	// The net/http.Server doesn't support h2c (unencrypted
	// HTTP/2) built-in.  Since we want to have gRPC and plain
	// HTTP/1 on the same unencrypted port, need h2c.
	// Fortunately, x/net has an h2c implementation we can use.
	return h2c.NewHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.ProtoMajor == 2 && strings.HasPrefix(r.Header.Get("Content-Type"), "application/grpc") {
			grpcServer.ServeHTTP(w, r)
		} else {
			path := r.URL.Path

			// reserve the /_/sys/* path for future internal "system" paths.
			if strings.HasPrefix(path, "/_/sys/healthz") || strings.HasPrefix(path, "/_/sys/readyz") {
				healthy := probe.Check()
				if healthy {
					w.WriteHeader(http.StatusOK)
				} else {
					w.WriteHeader(http.StatusInternalServerError)
				}
			} else {
				http.NotFound(w, r)
			}
		}
	}), &http2.Server{}), nil
}
