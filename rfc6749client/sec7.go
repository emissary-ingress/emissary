package rfc6749client

import (
	"net/http"

	"github.com/pkg/errors"
)

// AuthorizationForResourceRequest returns a set of HTTP header fields
// to inject in to HTTP requests to the Resource Server, in order to
// authorize the requests, per §7.1.
//
// This should be called separately for each outgoing request.
//
// BUG(lukeshu) AuthorizationForResourceRequest is rather simplistic.
// It would be insane to allow it to mutate the body, but being able
// to read the body would be useful for things like
// <https://tools.ietf.org/html/draft-hammer-oauth-v2-mac-token-05>.
// The shape that the API has taken is somewhat based on the API that
// Envoy ext_auth exposes; which is a very different shape than if I
// desiging this to be a "plain" Go library that I didn't know was
// being used to implement an Envoy filter.
func (token TokenSuccessResponse) AuthorizationForResourceRequest() (http.Header, error) {
	typeDriver, ok := accessTokenTypesRegistry[token.TokenType]
	if !ok {
		return nil, errors.Errorf("unsupported token_type: %q", token.TokenType)
	}
	return typeDriver.AuthorizationForResourceRequest()
}

// TODO: Do something here about looking up error codes.
