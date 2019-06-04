// Package rfc6750 registeres Bearer Token error codes for clients and
// resource servers.
package rfc6750

import (
	"github.com/datawire/liboauth2/common/rfc6749"
)

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

// §6.2.
func init() {
	rfc6749.ExtensionError{
		Name:                   "invalid_request",
		UsageLocations:         []rfc6749.ErrorUsageLocation{rfc6749.ResourceAccessErrorResponse},
		RelatedExtension:       "Bearer access token type",
		ChangeController:       "IETF",
		SpecificationDocuments: []string{"RFC 6750"},
		Meaning:                errorMeanings["invalid_request"],
	}.Register()
	rfc6749.ExtensionError{
		Name:                   "invalid_token",
		UsageLocations:         []rfc6749.ErrorUsageLocation{rfc6749.ResourceAccessErrorResponse},
		RelatedExtension:       "Bearer access token type",
		ChangeController:       "IETF",
		SpecificationDocuments: []string{"RFC 6750"},
		Meaning:                errorMeanings["invalid_token"],
	}.Register()
	rfc6749.ExtensionError{
		Name:                   "insufficient_scope",
		UsageLocations:         []rfc6749.ErrorUsageLocation{rfc6749.ResourceAccessErrorResponse},
		RelatedExtension:       "Bearer access token type",
		ChangeController:       "IETF",
		SpecificationDocuments: []string{"RFC 6750"},
		Meaning:                errorMeanings["insufficient_scope"],
	}.Register()
}
