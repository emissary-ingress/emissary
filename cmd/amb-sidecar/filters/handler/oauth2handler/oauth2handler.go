package oauth2handler

import (
	"context"
	"crypto/rsa"
	"net/http"

	"github.com/mediocregopher/radix.v2/pool"
	"github.com/pkg/errors"

	crd "github.com/datawire/apro/apis/getambassador.io/v1beta2"
	"github.com/datawire/apro/cmd/amb-sidecar/filters/handler/httpclient"
	"github.com/datawire/apro/cmd/amb-sidecar/filters/handler/middleware"
	"github.com/datawire/apro/cmd/amb-sidecar/filters/handler/oauth2handler/client"
	"github.com/datawire/apro/cmd/amb-sidecar/filters/handler/oauth2handler/discovery"
	"github.com/datawire/apro/cmd/amb-sidecar/filters/handler/oauth2handler/resourceserver"
	"github.com/datawire/apro/lib/filterapi"
)

// OAuth2Filter looks up the appropriate Tenant and Rule objects from
// the CRD Controller, and validates the signed JWT tokens when
// present in the request.  If the request Path is "/callback", it
// validates IDP requests and handles code exchange flow.
type OAuth2Filter struct {
	PrivateKey *rsa.PrivateKey
	PublicKey  *rsa.PublicKey
	RedisPool  *pool.Pool
	QName      string
	Spec       crd.FilterOAuth2
	Arguments  crd.FilterOAuth2Arguments
}

func (f *OAuth2Filter) Filter(ctx context.Context, request *filterapi.FilterRequest) (filterapi.FilterResponse, error) {
	logger := middleware.GetLogger(ctx)
	httpClient := httpclient.NewHTTPClient(logger, f.Spec.MaxStale, f.Spec.InsecureTLS)

	discovered, err := discovery.Discover(httpClient, f.Spec, logger)
	if err != nil {
		return middleware.NewErrorResponse(ctx, http.StatusBadGateway,
			errors.Wrap(err, "OIDC-discovery"), nil), nil
	}

	redisClient, err := f.RedisPool.Get()
	if err != nil {
		return middleware.NewErrorResponse(ctx, http.StatusBadGateway,
			errors.Wrap(err, "Redis"), nil), nil
	}
	defer f.RedisPool.Put(redisClient)

	oauth2client := &client.OAuth2Client{
		QName:     f.QName,
		Spec:      f.Spec,
		Arguments: f.Arguments,

		ResourceServer: &resourceserver.OAuth2ResourceServer{
			Spec:      f.Spec,
			Arguments: f.Arguments,
		},

		PrivateKey: f.PrivateKey,
		PublicKey:  f.PublicKey,
	}

	return oauth2client.Filter(ctx, logger, httpClient, discovered, redisClient, request), nil
}
