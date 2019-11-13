package handler

import (
	// 3rd-party libraries
	"github.com/mediocregopher/radix.v2/pool"
	"github.com/pkg/errors"

	// 1st-party libraries
	"github.com/datawire/ambassador/pkg/dlog"

	// internal libraries
	"github.com/datawire/apro/cmd/amb-sidecar/filters/controller"
	"github.com/datawire/apro/cmd/amb-sidecar/filters/handler/secret"
	"github.com/datawire/apro/cmd/amb-sidecar/limiter"
	"github.com/datawire/apro/cmd/amb-sidecar/types"
	"github.com/datawire/apro/lib/licensekeys"

	// k8s clients
	k8sClientCoreV1 "k8s.io/client-go/kubernetes/typed/core/v1"

	// k8s types
	crd "github.com/datawire/apro/apis/getambassador.io/v1beta2"
)

// Handler returns an app handler that should be consumed by an HTTP server.
func NewFilterMux(
	config types.Config,
	logger dlog.Logger,
	controller *controller.Controller,
	secretsGetter k8sClientCoreV1.SecretsGetter,
	redisPool *pool.Pool,
	limiter *limiter.LimiterImpl,
) (*FilterMux, error) {
	privKey, pubKey, err := secret.GetKeyPair(config, secretsGetter)
	if err != nil {
		// this is non fatal (mostly just to facilitate local dev); don't `return err`
		logger.Errorln("OAuth2 Filters with grantType=AuthorizationCode will not work correctly:", err)
	}

	// Error should never be set due to hardcoded enums
	// but if it is make it break hard.
	aesAuthLimiter, err := limiter.CreateRateLimiter(&licensekeys.LimitAuthFilterService)
	if err != nil {
		return nil, errors.Wrap(err, "limiter")
	}

	filterMux := &FilterMux{
		DefaultRule: &crd.Rule{
			Filters: nil,
		},
		Controller:      controller,
		PrivateKey:      privKey,
		PublicKey:       pubKey,
		Logger:          logger,
		RedisPool:       redisPool,
		AuthRateLimiter: aesAuthLimiter,
	}
	return filterMux, nil
}
