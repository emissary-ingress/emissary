package rfc6749

import (
	"io"
	"net/http"

	"github.com/pkg/errors"

	"github.com/datawire/liboauth2/common/rfc6749"
)

// AuthorizationForResourceRequest returns a set of HTTP header fields
// to inject in to HTTP requests to the Resource Server, in order to
// authorize the requests, per §7.1.
//
// This should be called separately for each outgoing request.
func (token TokenResponse) AuthorizationForResourceRequest(getBody func() io.Reader) (http.Header, error) {
	typeDriver := rfc6749.GetAccessTokenTypeClientDriver(token.TokenType)
	if typeDriver == nil {
		return nil, errors.Errorf("unsupported token_type: %q", token.TokenType)
	}
	var body io.Reader
	if typeDriver.NeedsBody() {
		body = getBody()
	}
	return typeDriver.AuthorizationForResourceRequest(token.AccessToken, body)
}

// GetResourceAccessErrorMeaning returns a human-readable meaning of
// an error code in an error response from a resource access request,
// per §7.2.  Returns an empty string for unknown error codes.
func GetResourceAccessErrorMeaning(errorCodeName string) string {
	ecode := rfc6749.GetResourceAccessError(errorCodeName)
	if ecode == nil {
		return ""
	}
	return ecode.Meaning()
}
