// Package rfc6750 provides Bearer Token support for OAuth 2.0 Clients.
package rfc6750

import (
	"fmt"
	"io"
	"net/http"
	"net/textproto"
	"net/url"
	"strings"

	"github.com/pkg/errors"

	"github.com/datawire/liboauth2/client/rfc6749"
	"github.com/datawire/liboauth2/syntax/rfc7235"

	// Register error codes shared between client and resourceserver.
	_ "github.com/datawire/liboauth2/common/rfc6750"
)

// AddToHeader adds a Bearer Token to an HTTP request header through the (RFC 7235, formerly RFC
// 2617) "Authorization" header field, as specified by §2.1.
func AddToHeader(token string, header http.Header) {
	header.Set("Authorization", "Bearer "+token)
}

// AddToBody adds a Bearer Token to an "application/xwww-form-urlencoded" request body, as specified
// by §2.2.
func AddToBody(token string, body url.Values) {
	body.Set("access_token", token)
}

// AuthorizationError represents the error response to an insufficiently authorized resource
// request, per §3.  This implements rfc6749.ResourceAccessErrorResponse.
type AuthorizationError struct {
	ParamRealm            *string       // MAY
	ParamScope            rfc6749.Scope // OPTIONAL
	ParamError            string        // SHOULD
	ParamErrorDescription string        // MAY
	ParamErrorURI         *url.URL      // MAY
}

func (ae *AuthorizationError) Error() string {
	params := []string{
		fmt.Sprintf("error=%q", ae.ParamError),
		fmt.Sprintf("error_description=%q", ae.ParamErrorDescription),
		fmt.Sprintf("error_uri=%q", ae.ParamErrorURI.String()),
	}
	if ae.ParamRealm != nil {
		params = append(params, fmt.Sprintf("realm=%q", *ae.ParamRealm))
	}
	if ae.ParamScope != nil {
		params = append(params, fmt.Sprintf("scope=%q", ae.ParamScope.String()))
	}
	return fmt.Sprintf("resource-access error-response: %s", strings.Join(params, " "))
}

// ErrorCode implements rfc6749.ResourceAccessErrorResponse.
func (ae *AuthorizationError) ErrorCode() string { return ae.ParamError }

// ErrorDescription implements rfc6749.ResourceAccessErrorResponse.
func (ae *AuthorizationError) ErrorDescription() string { return ae.ParamErrorDescription }

// ErrorURI implements rfc6749.ResourceAccessErrorResponse.
func (ae *AuthorizationError) ErrorURI() *url.URL { return ae.ParamErrorURI }

// ErrorFromErrorResponse inspects a Resource Access Response for the WWW-Authenticate header, which
// indicates an authorization failure, per §3.
func ErrorFromErrorResponse(resp *http.Response) (*AuthorizationError, error) {
	for _, challengeStr := range resp.Header[textproto.CanonicalMIMEHeaderKey("WWW-Authenticate")] {
		challenge, err := rfc7235.ParseChallenge(challengeStr)
		if !strings.EqualFold(challenge.AuthScheme, "Bearer") {
			continue
		}
		if err != nil {
			return nil, err
		}
		params, paramsOK := challenge.Body.(rfc7235.ChallengeParameters)
		if !paramsOK {
			return nil, errors.New("invalid Bearer challenge: used token68 syntax instead of auth-param syntax")
		}
		ret := &AuthorizationError{}
		for _, param := range params {
			switch {
			case strings.EqualFold(param.Key, "realm"):
				if ret.ParamRealm != nil {
					return nil, errors.New("invalid Bearer challenge: \"realm\" attribute MUST NOT appear more than once")
				}
				ret.ParamRealm = &param.Value
			case strings.EqualFold(param.Key, "scope"):
				if ret.ParamScope != nil {
					return nil, errors.New("invalid Bearer challenge: \"scope\" attribute MUST NOT appear more than once")
				}
				ret.ParamScope = rfc6749.ParseScope(param.Value)
			case strings.EqualFold(param.Key, "error"):
				ret.ParamError = param.Value
			case strings.EqualFold(param.Key, "error_description"):
				ret.ParamErrorDescription = param.Value
			case strings.EqualFold(param.Key, "error_uri"):
				u, err := url.Parse(param.Value)
				if err != nil {
					return nil, errors.Wrap(err, "invalid Bearer challenge: \"error_uri\" attribute is malformed")
				}
				ret.ParamErrorURI = u
			}
		}
		return ret, nil
	}
	return nil, nil
}

// OAuthProtocolExtension contains the information to register Bearer token support with an OAuth
// 2.0 Client, per §6.
//
// See ErrorFromResourceResponse for the behavior of the Client's .ErrorFromResourceResponse().
var OAuthProtocolExtension = rfc6749.ProtocolExtension{
	AccessTokenTypes: []rfc6749.AccessTokenType{
		{
			Name:                              "Bearer",
			AdditionalTokenEndpointParameters: nil,
			ChangeController:                  "IETF",
			SpecificationDocuments:            []string{"RFC 6750"},

			AuthorizationNeedsBody: false,
			AuthorizationForResourceRequest: func(token string, _ io.Reader) (http.Header, error) {
				ret := make(http.Header)
				AddToHeader(token, ret)
				return ret, nil
			},
			ErrorFromResourceResponse: func(resp *http.Response) (rfc6749.ResourceAccessErrorResponse, error) {
				// Silently convert from the struct *AuthorizationError to the
				// interface rfc6749.ResourceAccessErrorResponse.
				return ErrorFromErrorResponse(resp)
			},
		},
	},
}
