package rfc6749client

import (
	"net/http"
	"net/url"
	"strings"

	"github.com/pkg/errors"
)

// An AuthorizationCodeClient is Client that utilizes the
// "Authorization Code" Grant-type, as defined by §4.1.
type AuthorizationCodeClient struct {
	clientID              string
	authorizationEndpoint *url.URL
	tokenEndpoint         *url.URL
	clientAuthentication  ClientAuthenticationMethod
}

// NewAuthorizationCodeClient creates a new AuthorizationCodeClient as
// defined by §4.1.
func NewAuthorizationCodeClient(
	clientID string,
	authorizationEndpoint *url.URL,
	tokenEndpoint *url.URL,
	clientAuthentication ClientAuthenticationMethod,
) (*AuthorizationCodeClient, error) {
	if err := validateAuthorizationEndpointURI(authorizationEndpoint); err != nil {
		return nil, err
	}
	if err := validateTokenEndpointURI(tokenEndpoint); err != nil {
		return nil, err
	}
	ret := &AuthorizationCodeClient{
		clientID:              clientID,
		authorizationEndpoint: authorizationEndpoint,
		tokenEndpoint:         tokenEndpoint,
		clientAuthentication:  clientAuthentication,
	}
	return ret, nil
}

// AuthorizationRequest writes an HTTP response that directs the
// User-Agent to perform the Authorization Request, per §4.1.1.
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
//  - scopes: OPTIONAL.
//
//  - state: RECOMMENDED.
func (client *AuthorizationCodeClient) AuthorizationRequest(w http.ResponseWriter, r *http.Request, redirectURI *url.URL, scopes Scope, state string) {
	parameters := url.Values{
		"response_type": {"code"},
		"client_id":     {client.clientID},
	}
	if redirectURI != nil {
		err := validateRedirectionEndpointURI(redirectURI)
		if err != nil {
			err = errors.Wrap(err, "cannot build Authorization Request URI")
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		parameters.Set("redirect_uri", redirectURI.String())
	}
	if len(scopes) != 0 {
		parameters.Set("scope", scopes.String())
	}
	if state != "" {
		parameters.Set("state", state)
	}
	requestURI, err := buildAuthorizationRequestURI(client.authorizationEndpoint, parameters)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, requestURI.String(), http.StatusFound)
}

// ParseAuthorizationResponse parses the Authorization Response out
// from the HTTP request, as specified by §4.1.2.
//
// The returned response is either an
// AuthorizationCodeAuthorizationSuccessResponse or an
// AuthorizationCodeAuthorizationErrorResponse.  Either way, you
// should check that the .GetState() is valid before doing anything
// else with it.
//
// This should be called from the http.Handler for the Client's
// Redirection Endpoint.
func (client *AuthorizationCodeClient) ParseAuthorizationResponse(r *http.Request) (AuthorizationCodeAuthorizationResponse, error) {
	parameters := r.URL.Query()
	if errs := parameters["error"]; len(errs) > 0 {
		// §4.1.2.1 error
		ecodeData, ok := authorizationCodeAuthorizationErrorCodeData[errs[0]]
		if !ok {
			return nil, errors.Errorf("cannot parse error response: invalid error code: %q", errs[0])
		}
		var errorURI *url.URL
		if errorURIs := parameters["error_uri"]; len(errorURIs) > 0 {
			var err error
			errorURI, err = url.Parse(errorURIs[0])
			if err != nil {
				return nil, errors.Wrap(err, "cannot parse error response: invalid error_uri")
			}
		}
		return AuthorizationCodeAuthorizationErrorResponse{
			Error:            ecodeData.Self,
			ErrorDescription: parameters.Get("error_description"),
			ErrorURI:         errorURI,
		}, nil
	}
	// §4.1.2 success
	codes := parameters["code"]
	if len(codes) == 0 {
		return nil, errors.New("cannot parse response: missing required \"code\" parameter")
	}
	return AuthorizationCodeAuthorizationSuccessResponse{
		Code:  codes[0],
		State: parameters.Get("state"),
	}, nil
}

type AuthorizationCodeAuthorizationResponse interface {
	isAuthorizationCodeAuthorizationResponse()
	GetState() string
}

type AuthorizationCodeAuthorizationSuccessResponse struct {
	Code  string
	State string
}

func (r AuthorizationCodeAuthorizationSuccessResponse) isAuthorizationCodeAuthorizationResponse() {}
func (r AuthorizationCodeAuthorizationSuccessResponse) GetState() string                          { return r.State }

type AuthorizationCodeAuthorizationErrorResponse struct {
	Error AuthorizationCodeAuthorizationErrorCode

	// OPTIONAL.  Human-readable ASCII providing additional
	// information.
	ErrorDescription string

	// OPTIONAL.  A URI identifying a human-readable web page with
	// information about the error.
	ErrorURI *url.URL

	// REQUIRED if a "state" parameter was present in the
	// Authorization Request.
	State string
}

func (r AuthorizationCodeAuthorizationErrorResponse) isAuthorizationCodeAuthorizationResponse() {}
func (r AuthorizationCodeAuthorizationErrorResponse) GetState() string                          { return r.State }

// AuthorizationCodeAuthorizationErrorCode represents the error codes that may be
// returned by a failed "response_type=code" Authorization Request, as
// enumerated by §4.1.2.1.
type AuthorizationCodeAuthorizationErrorCode interface {
	isAuthorizationCodeAuthorizationErrorCode()
	String() string
	Description() string
}

type authorizationCodeAuthorizationErrorCode string

func (ecode authorizationCodeAuthorizationErrorCode) isAuthorizationCodeAuthorizationErrorCode() {}
func (ecode authorizationCodeAuthorizationErrorCode) String() string                             { return string(ecode) }
func (ecode authorizationCodeAuthorizationErrorCode) Description() string {
	return authorizationCodeAuthorizationErrorCodeData[string(ecode)].Description
}

var (
	AuthorizationCodeAuthorizationErrorInvalidRequest          AuthorizationCodeAuthorizationErrorCode = authorizationCodeAuthorizationErrorCode("invalid_request")
	AuthorizationCodeAuthorizationErrorUnauthorizedClient      AuthorizationCodeAuthorizationErrorCode = authorizationCodeAuthorizationErrorCode("unauthorized_client")
	AuthorizationCodeAuthorizationErrorAccessDenied            AuthorizationCodeAuthorizationErrorCode = authorizationCodeAuthorizationErrorCode("access_denied")
	AuthorizationCodeAuthorizationErrorUnsupportedResponseType AuthorizationCodeAuthorizationErrorCode = authorizationCodeAuthorizationErrorCode("unsupported_response_type")
	AuthorizationCodeAuthorizationErrorInvalidScope            AuthorizationCodeAuthorizationErrorCode = authorizationCodeAuthorizationErrorCode("invalid_scope")
	AuthorizationCodeAuthorizationErrorServerError             AuthorizationCodeAuthorizationErrorCode = authorizationCodeAuthorizationErrorCode("server_error")
	AuthorizationCodeAuthorizationErrorTemporarilyUnavailable  AuthorizationCodeAuthorizationErrorCode = authorizationCodeAuthorizationErrorCode("temporarily_unavailable")
)

var authorizationCodeAuthorizationErrorCodeData = map[string]struct {
	Self        AuthorizationCodeAuthorizationErrorCode
	Description string
}{
	"invalid_request": {
		Self: AuthorizationCodeAuthorizationErrorInvalidRequest,
		Description: "The request is missing a required parameter, includes an " +
			"invalid parameter value, includes a parameter more than " +
			"once, or is otherwise malformed."},

	"unauthorized_client": {
		Self: AuthorizationCodeAuthorizationErrorUnauthorizedClient,
		Description: "The client is not authorized to request an authorization " +
			"code using this method."},

	"access_denied": {
		Self: AuthorizationCodeAuthorizationErrorAccessDenied,
		Description: "The resource owner or authorization server denied the " +
			"request."},

	"unsupported_response_type": {
		Self: AuthorizationCodeAuthorizationErrorUnsupportedResponseType,
		Description: "The authorization server does not support obtaining an " +
			"authorization code using this method."},

	"invalid_scope": {
		Self:        AuthorizationCodeAuthorizationErrorInvalidScope,
		Description: "The requested scope is invalid, unknown, or malformed."},

	"server_error": {
		Self: AuthorizationCodeAuthorizationErrorServerError,
		Description: "The authorization server encountered an unexpected " +
			"condition that prevented it from fulfilling the request.  " +
			"(This error code is needed because a 500 Internal Server " +
			"Error HTTP status code cannot be returned to the client " +
			"via an HTTP redirect.)"},

	"temporarily_unavailable": {
		Self: AuthorizationCodeAuthorizationErrorTemporarilyUnavailable,
		Description: "The authorization server is currently unable to handle " +
			"the request due to a temporary overloading or maintenance " +
			"of the server.  (This error code is needed because a 503 " +
			"Service Unavailable HTTP status code cannot be returned " +
			"to the client via an HTTP redirect.)"},
}

// AccessToken talks to the Authorization Server to exchange an
// Authorization Code (obtained from `.ParseAuthorizationResponse()`)
// for an Access Token (and maybe a Refresh Token); submitting the
// request per §4.1.3, and handling the response per §4.1.4.
//
// redirectURI MUST match the redirectURI passed to
// .AuthorizationRequest().
//
// The returned response is either a TokenSuccessResponse or a
// TokenErrorResponse.
func (client *AuthorizationCodeClient) AccessToken(httpClient *http.Client, code string, redirectURI *url.URL) (TokenResponse, error) {
	if httpClient == nil {
		httpClient = http.DefaultClient
	}

	parameters := url.Values{
		"grant_type": {"authorization_code"},
		"code":       {"code"},
	}
	if redirectURI != nil {
		parameters.Set("redirect_uri", redirectURI.String())
	}
	if client.clientAuthentication == nil {
		parameters.Set("client_id", client.clientID)
	}

	req, err := http.NewRequest("POST", client.tokenEndpoint.String(), strings.NewReader(parameters.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	if client.clientAuthentication != nil {
		client.clientAuthentication(req)
	}

	res, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	return parseTokenResponse(res)
}
