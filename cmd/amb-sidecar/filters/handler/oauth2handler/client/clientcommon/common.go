package clientcommon

import (
	"context"
	"net/http"

	"github.com/pkg/errors"

	"github.com/datawire/ambassador/pkg/dlog"

	"github.com/datawire/apro/client/rfc6749"
	"github.com/datawire/apro/cmd/amb-sidecar/filters/handler/middleware"
	"github.com/datawire/apro/cmd/amb-sidecar/filters/handler/oauth2handler/discovery"
	"github.com/datawire/apro/cmd/amb-sidecar/filters/handler/oauth2handler/resourceserver"
	"github.com/datawire/apro/lib/filterapi"
	"github.com/datawire/apro/lib/filterapi/filterutil"
)

func HandleAuthenticatedProxyRequest(ctx context.Context, httpClient *http.Client, discovered *discovery.Discovered, request *filterapi.FilterRequest, authorization http.Header, scope rfc6749.Scope, resourceServer *resourceserver.OAuth2ResourceServer) filterapi.FilterResponse {
	addAuthorization := &filterapi.HTTPRequestModification{}
	for k, vs := range authorization {
		for _, v := range vs {
			addAuthorization.Header = append(addAuthorization.Header, &filterapi.HTTPHeaderReplaceValue{
				Key:   k,
				Value: v,
			})
		}
	}
	filterutil.ApplyRequestModification(request, addAuthorization)

	resourceResponse := resourceServer.Filter(ctx, dlog.GetLogger(ctx), httpClient, discovered, request, scope)
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
			)
		}
		return resourceResponse
	case *filterapi.HTTPRequestModification:
		return &filterapi.HTTPRequestModification{
			Header: append(addAuthorization.Header, resourceResponse.Header...),
		}
	default:
		panic(errors.Errorf("unknown resource server response type: %T", resourceResponse))
	}
}
