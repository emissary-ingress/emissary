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
	"github.com/datawire/apro/lib/filterapi"
	"github.com/datawire/apro/lib/filterapi/filterutil"
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

func (c *OAuth2Filter) Filter(ctx context.Context, request *filterapi.FilterRequest) (filterapi.FilterResponse, error) {
	logger := middleware.GetLogger(ctx)
	httpClient := httpclient.NewHTTPClient(logger, c.Spec.MaxStale, c.Spec.InsecureTLS)

	discovered, err := Discover(httpClient, c.Spec, logger)
	if err != nil {
		return middleware.NewErrorResponse(ctx, http.StatusBadGateway,
			errors.Wrap(err, "OIDC-discovery"), nil), nil
	}

	clientResponse := c.filterClient(ctx, logger, httpClient, discovered, request)
	switch clientResponse := clientResponse.(type) {
	case *filterapi.HTTPResponse:
		return clientResponse, nil
	case *filterapi.HTTPRequestModification:
		filterutil.ApplyRequestModification(request, clientResponse)
	default:
		panic(errors.Errorf("unexpexted filter response type %T", clientResponse))
	}

	resourceResponse := c.filterResourceServer(ctx, logger, httpClient, discovered, request)
	switch resourceResponse := resourceResponse.(type) {
	case *filterapi.HTTPResponse:
		if resourceResponse.StatusCode == http.StatusUnauthorized {
			// The upstream Resource Server returns 401 Unauthorized to the Client--the Client does NOT pass
			// 401 along to the User Agent; the User Agent is NOT using an RFC 7235-compatible
			// authentication scheme to talk to the Client; 401 would be inappropriate.
			//
			// Instead, wrap the 401 response in a 403 Forbidden response.
			return middleware.NewErrorResponse(ctx, http.StatusForbidden,
				errors.New("authorization rejected"),
				map[string]interface{}{
					"synthesized_upstream_response": resourceResponse,
				},
			), nil
		}
		return resourceResponse, nil
	case nil:
		// do nothing
	default:
		panic(errors.Errorf("unexpexted filter response type %T", resourceResponse))
	}

	return clientResponse, nil
}
