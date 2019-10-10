package handler

import (
	// 3rd-party libraries
	"github.com/mediocregopher/radix.v2/pool"
	"github.com/pkg/errors"

	// internal libraries
	"github.com/datawire/apro/cmd/amb-sidecar/filters/controller"
	secret "github.com/datawire/apro/cmd/amb-sidecar/filters/handler/secret"
	"github.com/datawire/apro/cmd/amb-sidecar/types"

	// k8s clients
	k8sClientCoreV1 "k8s.io/client-go/kubernetes/typed/core/v1"

	// k8s types
	crd "github.com/datawire/apro/apis/getambassador.io/v1beta2"
)

// Handler returns an app handler that should be consumed by an HTTP server.
func NewFilterMux(
	config types.Config,
	logger types.Logger,
	controller *controller.Controller,
	secretsGetter k8sClientCoreV1.SecretsGetter,
	redisPool *pool.Pool,
) (*FilterMux, error) {
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
		RedisPool:  redisPool,
	}
	return filterMux, nil
}
