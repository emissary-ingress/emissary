// Package rfc6750 provides Bearer Token support for OAuth 2.0 Resource Servers.
package rfc6750

import (
	"mime"
	"net/http"
	"net/url"
	"strings"

	"github.com/datawire/liboauth2/resourceserver/rfc6749"
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

// GetFromURI returns the Bearer Token extracted from a request URI query parameter, as specified by
// §2.3.  If there is no Bearer Token, it returns an empty string.
//
// If you do get the Bearer Token from the request URI, then "success (2XX status) responses to
// these requests SHOULD contain a Cache-Control header with the 'private' option"; it is up to you
// to include that option.
func GetFromURI(query url.Values) string {
	if len(query["access_token"]) != 1 {
		return ""
	}
	return query["access_token"][0]
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

// A TokenValidationFunc is a function that returns whether a given Bearer Token is valid.  If the
// token is determined to be valid, it must return (true, nil); if it is determined to be invalid,
// it must return (false, nil); if there is an error determining whether it is valid or invalid,
// then it must return an error.
type TokenValidationFunc func(token string) (valid bool, err error)

// OAuthProtocolExtension returns the information to register Bearer Token support with an OAuth 2.0
// ResourceServer, per §6.
//
// The supportBody and supportURI arguments identify whether to support extracting the Bearer Token
// from the request body and request URI respectively (in addition to being able to extract it from
// the request HTTP header, which is always supported).  Support for these is optional, and in the
// case of URI, actively discouraged.  If you do set supportURI=true, then "success (2XX status)
// responses to these requests SHOULD contain a Cache-Control header with the 'private' option"
// (§2.3); it is up to you to include that option.
func OAuthProtocolExtension(supportBody, supportURI bool, validate TokenValidationFunc) rfc6749.ProtocolExtension {
	return rfc6749.ProtocolExtension{
		AccessTokenTypes: []rfc6749.AccessTokenType{
			{
				Name:                              "Bearer",
				AdditionalTokenEndpointParameters: nil,
				ChangeController:                  "IETF",
				SpecificationDocuments:            []string{"RFC 6750"},

				ValidateAuthorization: func(req *http.Request) (bool, error) {
					token := GetFromHeader(req.Header)
					if token != "" {
						return validate(token)
					}
					var bodyErr error
					if supportBody {
						ct, _, err := mime.ParseMediaType(req.Header.Get("Content-Type"))
						if err != nil && ct == "application/x-www-form-urlencoded" {
							err := req.ParseForm()
							if err != nil {
								bodyErr = err
							} else if token := GetFromBody(req.PostForm); token != "" {
								return validate(token)
							}
						}
					}
					if supportURI {
						token := GetFromURI(req.URL.Query())
						if token != "" {
							return validate(token)
						}
					}
					if bodyErr != nil {
						return false, bodyErr
					}
					// TODO: maybe differentiate between different failure cases?
					return false, nil
				},
			},
		},
		ExtensionErrors: []rfc6749.ExtensionError{
			{
				Name:                   "invalid_request",
				UsageLocations:         []rfc6749.ErrorUsageLocation{rfc6749.LocationResourceAccessErrorResponse},
				RelatedExtension:       "Bearer access token type",
				ChangeController:       "IETF",
				SpecificationDocuments: []string{"RFC 6750"},

				Meaning: errorMeanings["invalid_request"],
			},

			{
				Name:                   "invalid_token",
				UsageLocations:         []rfc6749.ErrorUsageLocation{rfc6749.LocationResourceAccessErrorResponse},
				RelatedExtension:       "Bearer access token type",
				ChangeController:       "IETF",
				SpecificationDocuments: []string{"RFC 6750"},

				Meaning: errorMeanings["invalid_token"],
			},

			{
				Name:                   "insufficient_scope",
				UsageLocations:         []rfc6749.ErrorUsageLocation{rfc6749.LocationResourceAccessErrorResponse},
				RelatedExtension:       "Bearer access token type",
				ChangeController:       "IETF",
				SpecificationDocuments: []string{"RFC 6750"},

				Meaning: errorMeanings["insufficient_scope"],
			},
		},
	}
}
