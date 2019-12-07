// Package rfc6750 provides Bearer Token support for OAuth 2.0 Resource Servers.
package rfc6750

import (
	"fmt"
	"mime"
	"net/http"
	"net/url"
	"strings"

	"github.com/pkg/errors"

	"github.com/datawire/apro/common/rfc7235"
	"github.com/datawire/apro/resourceserver/rfc6749"
)

// GetFromHeader returns the Bearer Token extracted from an HTTP request header, as specified by
// §2.1.  If there is no Bearer Token, it returns an empty string and no error.  A valid Bearer
// Token is never empty.
func GetFromHeader(header http.Header) (string, error) {
	str := header.Get("Authorization")
	if str == "" {
		return "", nil
	}
	credentials, err := rfc7235.ParseCredentials(str)
	if err != nil {
		return "", errors.Wrap(err, "invalid Authorization header")
	}
	if !strings.EqualFold(credentials.AuthScheme, "Bearer") {
		return "", nil
	}
	token, tokenOK := credentials.Body.(rfc7235.CredentialsLegacy)
	if !tokenOK {
		return "", errors.New("invalid Bearer credentials: used auth-param syntax instead of token68 syntax")
	}
	return token.String(), nil
}

// GetFromBody returns the Bearer Token extracted from an "application/x-www-form-urlencoded"
// request body, as specified by §2.2.  If there is no Bearer Token, it returns an empty string and
// no error.  A valid Bearer Token is never empty.
func GetFromBody(req *http.Request) (string, error) {
	ct, _, err := mime.ParseMediaType(req.Header.Get("Content-Type"))
	if err != nil || ct != "application/x-www-form-urlencoded" {
		return "", nil
	}
	if err := req.ParseForm(); err != nil {
		return "", err
	}
	switch len(req.PostForm["access_token"]) {
	case 0:
		return "", nil
	case 1:
		token := req.PostForm["access_token"][0]
		if token == "" {
			return "", errors.New("invalid Bearer credentials: empty (but set) access_token body parameter")
		}
		return token, nil
	default:
		return "", errors.New("invalid Bearer credentials: repeated access_token body parameter")
	}
}

// GetFromURI returns the Bearer Token extracted from a request URI query parameter, as specified by
// §2.3.  If there is no Bearer Token, it returns an empty string and no error.  A valid Bearer
// Token is never empty.
//
// If you do get the Bearer Token from the request URI, then "success (2XX status) responses to
// these requests SHOULD contain a Cache-Control header with the 'private' option"; it is up to you
// to include that option.
func GetFromURI(query url.Values) (string, error) {
	switch len(query["access_token"]) {
	case 0:
		return "", nil
	case 1:
		token := query["access_token"][0]
		if token == "" {
			return "", errors.New("invalid Bearer credentials: empty (but set) access_token query parameter")
		}
		return token, nil
	default:
		return "", errors.New("invalid Bearer credentials: repeated access_token query parameter")
	}
}

type AuthorizationValidator struct {
	// SupportBody and SupportURI identify whether to support extracting the Bearer Token from
	// the request body and request URI respectively (in addition to being able to extract it
	// from the request HTTP header, which is always supported).  Support for these is optional,
	// and in the case of URI, actively discouraged.  If you do set SupportURI=true, then
	// "success (2XX status) responses to these requests SHOULD contain a Cache-Control header
	// with the 'private' option" (§2.3); it is up to you to include that option.
	SupportBody bool
	SupportURI  bool

	// Realm is the realm (if any) to self-identify as in WWW-Authenticate challenges.
	Realm string

	// TokenValidationFunc is a function that returns whether a given Bearer Token is valid.  If
	// the token is determined to be valid, it must return (scope, nil, nil) where (scope) is
	// the scope of the token.  If the token is determined to be invalid, it must return (nil,
	// reason, nil).  If there is an error determining whether the token is valid or invalid,
	// then it must return (nil, nil, reason).
	TokenValidationFunc func(token string) (scope rfc6749.Scope, reasonInvalid, serverError error)

	// RequiredScope is this minimum scope an Access Token must have to authorize a request.
	RequiredScope rfc6749.Scope
}

// Mash-up of §2 and §3
func (v *AuthorizationValidator) get(req *http.Request) (string, error) {
	var token, _token string
	var cnt uint
	var err error

	_token, err = GetFromHeader(req.Header)
	if err != nil {
		return "", err
	}
	if _token != "" {
		token = _token
		cnt++
	}

	if v.SupportURI {
		_token, err = GetFromURI(req.URL.Query())
		if err != nil {
			return "", err
		}
		if _token != "" {
			token = _token
			cnt++
		}
	}

	if v.SupportBody {
		_token, err = GetFromBody(req)
		if err != nil {
			return "", err
		}
		if _token != "" {
			token = _token
			cnt++
		}
	}

	switch cnt {
	case 0:
		return "", nil
	case 1:
		return token, nil
	default:
		return "", errors.New("invalid Bearer credentials: access token provided with multiple methods")
	}
}

// §3
func (v *AuthorizationValidator) fmtChallenge(challengeParams rfc7235.ChallengeParameters) rfc7235.Challenge {
	if v.Realm != "" {
		challengeParams = append(challengeParams,
			rfc7235.AuthParam{Key: "realm", Value: v.Realm})
	}
	return rfc7235.Challenge{
		AuthScheme: "Bearer",
		Body:       challengeParams,
	}
}

// AuthorizationError represents the error response to an insufficiently authorized resource
// request, per §3.
type AuthorizationError struct {
	HTTPStatusCode int
	Challenge      rfc7235.Challenge
	err            error
}

func (e *AuthorizationError) String() string {
	return fmt.Sprintf("HTTP %d / WWW-Authorize: %s", e.HTTPStatusCode, e.Challenge)
}

func (e *AuthorizationError) Error() string {
	if e.err != nil {
		return e.err.Error()
	}
	if params, paramsOK := e.Challenge.Body.(rfc7235.ChallengeParameters); paramsOK {
		for _, param := range params {
			if param.Key == "error_description" {
				return param.Value
			}
		}
	}
	return e.String()
}

// ValidateAuthorization inspects a request received and decides whether to authorize it.  If the
// request is authorized, then nil is returned.  If the request is not authorized, then an error of
// type *AuthoriationError is returned.  Errors of other types indicate that there was an error in
// deciding whether to authorize the request.
func (v *AuthorizationValidator) ValidateAuthorization(req *http.Request) error {
	token, err := v.get(req)
	if err != nil {
		return &AuthorizationError{
			HTTPStatusCode: http.StatusBadRequest,
			Challenge: v.fmtChallenge(rfc7235.ChallengeParameters{
				{Key: "error", Value: "invalid_request"},
				{Key: "error_description", Value: err.Error()},
			}),
		}
	}
	if token == "" {
		return &AuthorizationError{
			HTTPStatusCode: http.StatusUnauthorized,
			Challenge:      v.fmtChallenge(rfc7235.ChallengeParameters{}),
			err:            errors.New("no Bearer token"),
		}
	}

	actualScope, invalidErr, serverErr := v.TokenValidationFunc(token)
	if serverErr != nil {
		return serverErr
	}
	if invalidErr != nil {
		return &AuthorizationError{
			HTTPStatusCode: http.StatusUnauthorized,
			Challenge: v.fmtChallenge(rfc7235.ChallengeParameters{
				{Key: "error", Value: "invalid_token"},
				{Key: "error_description", Value: invalidErr.Error()},
			}),
		}
	}

	var missing []string
	for scopeValue := range v.RequiredScope {
		if _, ok := actualScope[scopeValue]; !ok {
			missing = append(missing, scopeValue)
		}
	}
	switch len(missing) {
	case 0:
		return nil
	case 1:
		err = errors.Errorf("missing required scope value: %q", missing[0])
	default:
		err = errors.Errorf("missing required scope values: %q", missing)
	}
	return &AuthorizationError{
		HTTPStatusCode: http.StatusForbidden,
		Challenge: v.fmtChallenge(rfc7235.ChallengeParameters{
			{Key: "error", Value: "insufficient_scope"},
			{Key: "error_description", Value: err.Error()},
			{Key: "scope", Value: v.RequiredScope.String()},
		}),
	}
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

// OAuthProtocolExtension returns the information to register Bearer Token support with an OAuth 2.0
// ResourceServer, per §6.
//
// If you do set validator.SupportURI=true, then "success (2XX status) responses to these requests
// SHOULD contain a Cache-Control header with the 'private' option" (§2.3); it is up to you to
// include that option.
func OAuthProtocolExtension(validator *AuthorizationValidator) rfc6749.ProtocolExtension {
	return rfc6749.ProtocolExtension{
		AccessTokenTypes: []rfc6749.AccessTokenType{
			{
				Name:                              "Bearer",
				AdditionalTokenEndpointParameters: nil,
				ChangeController:                  "IETF",
				SpecificationDocuments:            []string{"RFC 6750"},

				ValidateAuthorization: validator.ValidateAuthorization,
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
