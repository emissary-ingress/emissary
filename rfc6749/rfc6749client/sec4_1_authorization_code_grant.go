package rfc6749client

import (
	"net/http"
	"net/url"

	"github.com/pkg/errors"

	"github.com/datawire/liboauth2/rfc6749/rfc6749registry"
)

// An AuthorizationCodeClient is a Client that utilizes the
// "Authorization Code" Grant-type, as defined by §4.1.
type AuthorizationCodeClient struct {
	clientID              string
	authorizationEndpoint *url.URL
	explicitClient
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
		explicitClient: explicitClient{
			tokenEndpoint:        tokenEndpoint,
			clientAuthentication: clientAuthentication,
		},
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
//  - scope: OPTIONAL.
//
//  - state: RECOMMENDED.
func (client *AuthorizationCodeClient) AuthorizationRequest(
	w http.ResponseWriter, r *http.Request,
	redirectURI *url.URL, scope Scope, state string,
) {
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
	if len(scope) != 0 {
		parameters.Set("scope", scope.String())
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
		var errorURI *url.URL
		if errorURIs := parameters["error_uri"]; len(errorURIs) > 0 {
			var err error
			errorURI, err = url.Parse(errorURIs[0])
			if err != nil {
				return nil, errors.Wrap(err, "cannot parse error response: invalid error_uri")
			}
		}
		return AuthorizationCodeAuthorizationErrorResponse{
			Error:            errs[0],
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

// AuthorizationCodeAuthorizationResponse encapsulates the possible
// responses to an Authorization Request in the Authorization Code
// flow.
//
// This is implemented by
// AuthorizationCodeAuthorizationSuccessResponse and an
// AuthorizationCodeAuthorizationErrorResponse.
type AuthorizationCodeAuthorizationResponse interface {
	isAuthorizationCodeAuthorizationResponse()
	GetState() string
}

// An AuthorizationCodeAuthorizationSuccessResponse is a successful
// response to an Authorization Request in the Authorization Code
// flow, as defined in §4.1.2.
type AuthorizationCodeAuthorizationSuccessResponse struct {
	Code  string
	State string
}

func (r AuthorizationCodeAuthorizationSuccessResponse) isAuthorizationCodeAuthorizationResponse() {}

// GetState returns the state parameter (if any) included in the response.
func (r AuthorizationCodeAuthorizationSuccessResponse) GetState() string { return r.State }

// An AuthorizationCodeAuthorizationErrorResponse is an error response
// to an Authorization Request in the Authorization Code flow, as
// defined in §4.1.2.1.
type AuthorizationCodeAuthorizationErrorResponse struct {
	Error string

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

// GetState returns the state parameter (if any) included in the response.
func (r AuthorizationCodeAuthorizationErrorResponse) GetState() string { return r.State }

// ErrorMeaning returns a human-readable meaning of the .Error code.
// Returns an empty string for unknown error codes.
func (r AuthorizationCodeAuthorizationErrorResponse) ErrorMeaning() string {
	ecode := rfc6749registry.GetAuthorizationCodeGrantError(r.Error)
	if ecode == nil {
		return ""
	}
	return ecode.Meaning()
}

func newAuthorizationCodeError(name, meaning string) {
	rfc6749registry.ExtensionError{
		Name:    name,
		Meaning: meaning,
		UsageLocations: []rfc6749registry.ErrorUsageLocation{
			rfc6749registry.AuthorizationCodeGrantErrorResponse,
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
// redirectURI MUST match the redirectURI passed to
// .AuthorizationRequest().
//
// The returned response is either a TokenSuccessResponse or a
// TokenErrorResponse.
func (client *AuthorizationCodeClient) AccessToken(httpClient *http.Client, code string, redirectURI *url.URL) (TokenResponse, error) {
	parameters := url.Values{
		"grant_type": {"authorization_code"},
		"code":       {code},
	}
	if redirectURI != nil {
		parameters.Set("redirect_uri", redirectURI.String())
	}
	if client.explicitClient.clientAuthentication == nil {
		parameters.Set("client_id", client.clientID)
	}

	return client.explicitClient.postForm(httpClient, parameters)
}
