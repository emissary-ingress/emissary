package rfc6749client

import (
	"io"
	"net/http"

	"github.com/pkg/errors"

	"github.com/datawire/liboauth2/rfc6749/rfc6749registry"
)

// AuthorizationForResourceRequest returns a set of HTTP header fields
// to inject in to HTTP requests to the Resource Server, in order to
// authorize the requests, per §7.1.
//
// This should be called separately for each outgoing request.
func (token TokenSuccessResponse) AuthorizationForResourceRequest(getBody func() io.Reader) (http.Header, error) {
	typeDriver := rfc6749registry.GetAccessTokenTypeClientDriver(token.TokenType)
	if typeDriver == nil {
		return nil, errors.Errorf("unsupported token_type: %q", token.TokenType)
	}
	var body io.Reader
	if typeDriver.NeedsBody() {
		body = getBody()
	}
	return typeDriver.AuthorizationForResourceRequest(body)
}

// GetResourceAccessErrorMeaning returns a humman-readable meaning of
// an error code in an error response from a resource access request,
// per §7.2.  Returns an empty string for unknown error codes.
func GetResourceAccessErrorMeaning(errorCodeName string) string {
	ecode := rfc6749registry.GetResourceAccessError(errorCodeName)
	if ecode == nil {
		return ""
	}
	return ecode.Meaning()
}
