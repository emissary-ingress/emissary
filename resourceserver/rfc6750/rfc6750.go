// Package rfc6750 provides Bearer Token support for the OAuth 2.0
// framework.
package rfc6750

import (
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/pkg/errors"

	"github.com/datawire/liboauth2/rfc6749/rfc6749registry"
)

// AddToHeader adds a Bearer Token to an HTTP request header through
// the (RFC 7235, formerly RFC 2617) "Authorization" header field, as
// specified by §2.1.
func AddToHeader(token string, header http.Header) {
	header.Set("Authorization", "Bearer "+token)
}

// GetFromHeader returns the Bearer Token in an HTTP request header as
// specified by §2.1.  If there is no Bearer Token, it returns an
// empty string.
func GetFromHeader(header http.Header) string {
	valueParts := strings.SplitN(header.Get("Authorization"), " ", 2)
	if len(valueParts) != 2 || !strings.EqualFold(valueParts[0], "Bearer") {
		return ""
	}
	return valueParts[1]
}

// AddToBody adds a Bearer Token to an
// "application/xwww-form-urlencoded" request body, as specified by
// §2.2.
func AddToBody(token string, body url.Values) {
	body.Set("access_token", token)
}

// GetFromBody returns the Bearer Token in an
// "application/x-www-form-urlencoded" request body, as specified by
// §2.2..  If there is no Bearer Token, it returns an empty string.
func GetFromBody(body url.Values) string {
	if len(body["access_token"]) != 1 {
		return ""
	}
	return body["access_token"][0]
}

// §3.1.
var errorMeanings = map[string]string{
	"invalid_request": "" +
		"The request is missing a required parameter, includes an " +
		"unsupported parameter or parameter value, repeats the same " +
		"parameter, uses more than one method for including an access " +
		"token, or is otherwise malformed.  The resource server SHOULD " +
		"respond with the HTTP 400 (Bad Request) status code.",

	"invalid_token": "" +
		"The access token provided is expired, revoked, malformed, or " +
		"invalid for other reasons.  The resource SHOULD respond with " +
		"the HTTP 401 (Unauthorized) status code.  The client MAY " +
		"request a new access token and retry the protected resource " +
		"request.",

	"insufficient_scope": "" +
		"The request requires higher privileges than provided by the " +
		"access token.  The resource server SHOULD respond with the HTTP " +
		"403 (Forbidden) status code and MAY include the \"scope\" " +
		"attribute with the scope necessary to access the protected " +
		"resource.",
}

// §6.
func init() {
	// §6.1.
	rfc6749registry.AccessTokenType{
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

	// §6.2.
	rfc6749registry.ExtensionError{
		Name:                   "invalid_request",
		UsageLocations:         []rfc6749registry.ErrorUsageLocation{rfc6749registry.ResourceAccessErrorResponse},
		RelatedExtension:       "Bearer access token type",
		ChangeController:       "IETF",
		SpecificationDocuments: []string{"RFC 6750"},
		Meaning:                errorMeanings["invalid_request"],
	}.Register()
	rfc6749registry.ExtensionError{
		Name:                   "invalid_token",
		UsageLocations:         []rfc6749registry.ErrorUsageLocation{rfc6749registry.ResourceAccessErrorResponse},
		RelatedExtension:       "Bearer access token type",
		ChangeController:       "IETF",
		SpecificationDocuments: []string{"RFC 6750"},
		Meaning:                errorMeanings["invalid_token"],
	}.Register()
	rfc6749registry.ExtensionError{
		Name:                   "insufficient_scope",
		UsageLocations:         []rfc6749registry.ErrorUsageLocation{rfc6749registry.ResourceAccessErrorResponse},
		RelatedExtension:       "Bearer access token type",
		ChangeController:       "IETF",
		SpecificationDocuments: []string{"RFC 6750"},
		Meaning:                errorMeanings["insufficient_scope"],
	}.Register()
}
