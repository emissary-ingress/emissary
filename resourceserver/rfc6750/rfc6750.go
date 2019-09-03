// Package rfc6750 provides Bearer Token support for OAuth 2.0 Resource Servers.
package rfc6750

import (
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/pkg/errors"

	"github.com/datawire/liboauth2/common/rfc6749"

	// Register error codes shared between client and resourceserver.
	_ "github.com/datawire/liboauth2/common/rfc6750"
)

// GetFromHeader returns the Bearer Token extracted from an HTTP request header, as specified by
// §2.1.  If there is no Bearer Token, it returns an empty string.
func GetFromHeader(header http.Header) string {
	valueParts := strings.SplitN(header.Get("Authorization"), " ", 2)
	if len(valueParts) != 2 || !strings.EqualFold(valueParts[0], "Bearer") {
		return ""
	}
	return valueParts[1]
}

// GetFromBody returns the Bearer Token extracted from an "application/x-www-form-urlencoded"
// request body, as specified by §2.2.  If there is no Bearer Token, it returns an empty string.
func GetFromBody(body url.Values) string {
	if len(body["access_token"]) != 1 {
		return ""
	}
	return body["access_token"][0]
}

// §6.1.
func init() {
	rfc6749.AccessTokenType{
		Name:                              "Bearer",
		AdditionalTokenEndpointParameters: nil,
		ChangeController:                  "IETF",
		SpecificationDocuments:            []string{"RFC 6750"},

		ResourceServerNeedsBody: false,
		ResourceServerValidateAuthorization: func(header http.Header, _ io.Reader) (bool, error) {
			token := GetFromHeader(header)
			if token == "" {
				// TODO: maybe differentiate between different failure cases?
				return false, nil
			}
			// TODO: Stub this out somehow
			return false, errors.New("not implemented")
		},
	}.Register()
}
