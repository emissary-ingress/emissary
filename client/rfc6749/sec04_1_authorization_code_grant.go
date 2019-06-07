package rfc6749

import (
	"fmt"
	"net/http"
	"net/url"

	"github.com/pkg/errors"

	"github.com/datawire/liboauth2/common/rfc6749"
)

// An AuthorizationCodeClient is a Client that utilizes the
// "Authorization Code" Grant-type, as defined by §4.1.
type AuthorizationCodeClient struct {
	clientID              string
	authorizationEndpoint *url.URL

	explicitClient
	accessTokenTypeRegistry
}

// NewAuthorizationCodeClient creates a new AuthorizationCodeClient as
// defined by §4.1.
func NewAuthorizationCodeClient(
	clientID string,
	authorizationEndpoint *url.URL,
	tokenEndpoint *url.URL,
	clientAuthentication ClientAuthenticationMethod,
	httpClient *http.Client,
) (*AuthorizationCodeClient, error) {
	if err := validateAuthorizationEndpointURI(authorizationEndpoint); err != nil {
		return nil, err
	}
	if err := validateTokenEndpointURI(tokenEndpoint); err != nil {
		return nil, err
	}
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	ret := &AuthorizationCodeClient{
		clientID:              clientID,
		authorizationEndpoint: authorizationEndpoint,
		explicitClient: explicitClient{
			tokenEndpoint:        tokenEndpoint,
			clientAuthentication: clientAuthentication,
			httpClient:           httpClient,
		},
	}
	return ret, nil
}

// AuthorizationCodeClientSessionData is the session data that must be
// persisted between requests when using an AuthorizationCodeClient.
type AuthorizationCodeClientSessionData struct {
	Request struct {
		RedirectURI *url.URL
		Scope       Scope
		State       string
	}
	CurrentAccessToken *accessTokenData
	isDirty            bool
}

func (session AuthorizationCodeClientSessionData) accessToken() *accessTokenData {
	return session.CurrentAccessToken
}
func (session AuthorizationCodeClientSessionData) setDirty() { session.isDirty = true }

// IsDirty indicates whether the session data has been mutated since
// that last time that it was unmarshaled.  This is only useful if you
// marshal it to and unmarshal it from an external datastore.
func (session AuthorizationCodeClientSessionData) IsDirty() bool { return session.isDirty }

// AuthorizationRequest returns an URI that the Client should direct
// the User-Agent to perform a GET request for, in order to perform an
// Authorization Request, per §4.1.1.
//
// OAuth arguments:
//
//  - redirectURI: OPTIONAL if exactly 1 complete Redirection Endpoint
//    was registered with the Authorization Server when registering
//    the Client.  If the Client was not registered with the
//    Authorization Server, it was registered with 0 Redirection
//    Endpoints, it was registered with a partial Redirection
//    Endpoint, or it was registered with more than 1 Redirection
//    Endpoint, then this argument is REQUIRED.
//
//  - scope: OPTIONAL.
//
//  - state: RECOMMENDED.
//
// The Client is free to use whichever redirection mechanisms it has
// available to it (perhaps a plain HTTP redirect, or perhaps
// something fancy with JavaScript).  Note that if using an HTTP
// redirect, that 302 "Found" may or MAY NOT convert POST->GET; and
// that to reliably have the User-Agent perform a GET, one should use
// 303 "See Other" which MUST convert to GET.
func (client *AuthorizationCodeClient) AuthorizationRequest(redirectURI *url.URL, scope Scope, state string) (*url.URL, *AuthorizationCodeClientSessionData, error) {
	parameters := url.Values{
		"response_type": {"code"},
		"client_id":     {client.clientID},
	}
	if redirectURI != nil {
		err := validateRedirectionEndpointURI(redirectURI)
		if err != nil {
			return nil, nil, errors.Wrap(err, "cannot build Authorization Request URI")
		}
		parameters.Set("redirect_uri", redirectURI.String())
	}
	if len(scope) != 0 {
		parameters.Set("scope", scope.String())
	}
	if state != "" {
		parameters.Set("state", state)
	}

	session := &AuthorizationCodeClientSessionData{}
	session.Request.RedirectURI = redirectURI
	session.Request.Scope = scope
	session.Request.State = state
	session.isDirty = true

	u, err := buildAuthorizationRequestURI(client.authorizationEndpoint, parameters)
	if err != nil {
		return nil, nil, err
	}

	return u, session, nil
}

// ParseAuthorizationResponse parses the Authorization Response out
// from the HTTP request URL, as specified by §4.1.2.
//
// This should be called from the http.Handler for the Client's
// Redirection Endpoint.
//
// If the server sent a semantically valid error response, the
// returned error is of type AuthorizationCodeGrantErrorResponse.  On
// protocol errors, a different error type is returned.
func (client *AuthorizationCodeClient) ParseAuthorizationResponse(session *AuthorizationCodeClientSessionData, requestURL *url.URL) (authorizationCode string, err error) {
	parameters := requestURL.Query()

	// The "state" parameter is shared by both success and error
	// responses.  Let's check this early, to avoid unnecessary
	// resource usage.
	if parameters.Get("state") != session.Request.State {
		return "", errors.New("refusing to parse response: response state parameter does not match request state parameter; XSRF attack likely")
	}

	if errs := parameters["error"]; len(errs) > 0 {
		// §4.1.2.1 error
		var errorURI *url.URL
		if errorURIs := parameters["error_uri"]; len(errorURIs) > 0 {
			var err error
			errorURI, err = url.Parse(errorURIs[0])
			if err != nil {
				return "", errors.Wrap(err, "cannot parse error response: invalid error_uri")
			}
		}
		return "", AuthorizationCodeGrantErrorResponse{
			ErrorCode:        errs[0],
			ErrorDescription: parameters.Get("error_description"),
			ErrorURI:         errorURI,
		}
	}
	// §4.1.2 success
	codes := parameters["code"]
	if len(codes) == 0 {
		return "", errors.New("cannot parse response: missing required \"code\" parameter")
	}
	return codes[0], nil
}

// An AuthorizationCodeGrantErrorResponse is an error response
// to an Authorization Request in the Authorization Code flow, as
// defined in §4.1.2.1.
type AuthorizationCodeGrantErrorResponse struct {
	// REQUIRED.  A single ASCII error code.
	ErrorCode string

	// OPTIONAL.  Human-readable ASCII providing additional
	// information, used to assist the client developer.
	ErrorDescription string

	// OPTIONAL.  A URI identifying a human-readable web page with
	// information about the error, used to provide the client
	// developer with additional information.
	ErrorURI *url.URL
}

func (r AuthorizationCodeGrantErrorResponse) Error() string {
	ret := fmt.Sprintf("authorization code grant error response: error=%q", r.ErrorCode)
	if r.ErrorDescription != "" {
		ret = fmt.Sprintf("%s error_description=%q", ret, r.ErrorDescription)
	}
	if r.ErrorURI != nil {
		ret = fmt.Sprintf("%s error_uri=%q", ret, r.ErrorURI.String())
	}
	return ret
}

func newAuthorizationCodeError(name, meaning string) {
	rfc6749.ExtensionError{
		Name:    name,
		Meaning: meaning,
		UsageLocations: []rfc6749.ErrorUsageLocation{
			rfc6749.AuthorizationCodeGrantErrorResponse,
		},
	}.Register()
}

// These are the built-in error codes that may be present in an
// AuthorizationCodeAuthorizationErrorResponse, as enumerated in
// §4.1.2.1.  This set may be extended by extension error registry.
func init() {
	newAuthorizationCodeError("invalid_request", ""+
		"The request is missing a required parameter, includes an "+
		"invalid parameter value, includes a parameter more than "+
		"once, or is otherwise malformed.")

	newAuthorizationCodeError("unauthorized_client", ""+
		"The client is not authorized to request an authorization "+
		"code using this method.")

	newAuthorizationCodeError("access_denied", ""+
		"The resource owner or authorization server denied the "+
		"request.")

	newAuthorizationCodeError("unsupported_response_type", ""+
		"The authorization server does not support obtaining an "+
		"authorization code using this method.")

	newAuthorizationCodeError("invalid_scope", ""+
		"The requested scope is invalid, unknown, or malformed.")

	newAuthorizationCodeError("server_error", ""+
		"The authorization server encountered an unexpected "+
		"condition that prevented it from fulfilling the request.  "+
		"(This error code is needed because a 500 Internal Server "+
		"Error HTTP status code cannot be returned to the client "+
		"via an HTTP redirect.)")

	newAuthorizationCodeError("temporarily_unavailable", ""+
		"The authorization server is currently unable to handle "+
		"the request due to a temporary overloading or maintenance "+
		"of the server.  (This error code is needed because a 503 "+
		"Service Unavailable HTTP status code cannot be returned "+
		"to the client via an HTTP redirect.)")
}

// AccessToken talks to the Authorization Server to exchange an
// Authorization Code (obtained from `.ParseAuthorizationResponse()`)
// for an Access Token (and maybe a Refresh Token); submitting the
// request per §4.1.3, and handling the response per §4.1.4.
//
// The returned response is either a TokenSuccessResponse or a
// TokenErrorResponse.
func (client *AuthorizationCodeClient) AccessToken(session *AuthorizationCodeClientSessionData, authorizationCode string) (TokenResponse, error) {
	parameters := url.Values{
		"grant_type": {"authorization_code"},
		"code":       {authorizationCode},
	}
	if session.Request.RedirectURI != nil {
		parameters.Set("redirect_uri", session.Request.RedirectURI.String())
	}
	if client.explicitClient.clientAuthentication == nil {
		parameters.Set("client_id", client.clientID)
	}

	return client.postForm(parameters)
}
