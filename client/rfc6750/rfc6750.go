// Package rfc6750 provides Bearer Token support for OAuth 2.0
// Clients.
package rfc6750

import (
	"io"
	"net/http"
	"net/url"

	"github.com/datawire/liboauth2/common/rfc6749"

	// Register error codes shared between client and
	// resourceserver.
	_ "github.com/datawire/liboauth2/common/rfc6750"
)

// AddToHeader adds a Bearer Token to an HTTP request header through
// the (RFC 7235, formerly RFC 2617) "Authorization" header field, as
// specified by §2.1.
func AddToHeader(token string, header http.Header) {
	header.Set("Authorization", "Bearer "+token)
}

// AddToBody adds a Bearer Token to an
// "application/xwww-form-urlencoded" request body, as specified by
// §2.2.
func AddToBody(token string, body url.Values) {
	body.Set("access_token", token)
}

// §6.1.
func init() {
	rfc6749.AccessTokenType{
		Name:                              "Bearer",
		AdditionalTokenEndpointParameters: nil,
		ChangeController:                  "IETF",
		SpecificationDocuments:            []string{"RFC 6750"},

		ClientNeedsBody: false,
		ClientAuthorizationForResourceRequest: func(token string, _ io.Reader) (http.Header, error) {
			ret := make(http.Header)
			AddToHeader(token, ret)
			return ret, nil
		},
	}.Register()
}
