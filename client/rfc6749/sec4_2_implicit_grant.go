package rfc6749

import (
	"net/url"
	"strconv"
	"time"

	"github.com/pkg/errors"

	"github.com/datawire/liboauth2/common/rfc6749"
)

// An ImplicitClient is a Client that utilizes the "Implicit"
// Grant-type, as defined by §4.2.
type ImplicitClient struct {
	clientID              string
	authorizationEndpoint *url.URL
}

// NewImplicitClient creates a new ImplicitClient as defined by §4.2.
func NewImplicitClient(
	clientID string,
	authorizationEndpoint *url.URL,
) (*ImplicitClient, error) {
	if err := validateAuthorizationEndpointURI(authorizationEndpoint); err != nil {
		return nil, err
	}
	ret := &ImplicitClient{
		clientID:              clientID,
		authorizationEndpoint: authorizationEndpoint,
	}
	return ret, nil
}

// AuthorizationRequest returns an URI that the Client should direct
// the User-Agent to perform a GET request for, in order to perform an
// Authorization Request, per §4.2.1.
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
func (client *ImplicitClient) AuthorizationRequest(redirectURI *url.URL, scope Scope, state string) (*url.URL, error) {
	parameters := url.Values{
		"response_type": {"token"},
		"client_id":     {client.clientID},
	}
	if redirectURI != nil {
		err := validateRedirectionEndpointURI(redirectURI)
		if err != nil {
			return nil, errors.Wrap(err, "cannot build Authorization Request URI")
		}
		parameters.Set("redirect_uri", redirectURI.String())
	}
	if len(scope) != 0 {
		parameters.Set("scope", scope.String())
	}
	if state != "" {
		parameters.Set("state", state)
	}
	return buildAuthorizationRequestURI(client.authorizationEndpoint, parameters)
}

// ParseAccessTokenResponse parses the URI fragment that contains the
// Access Token Response, as specified by §4.2.2.
//
// The returned response is either an
// ImplicitAccessTokenSuccessResponse or an
// ImplicitAccessTokenErrorResponse.  Either way, you should check
// that the .GetState() is valid before doing anything else with it.
//
// The fragment is normally not accessible to the HTTP server.  You
// will need to use JavaScript in the user-agent to somehow get it to
// the server.
func (client *ImplicitClient) ParseAccessTokenResponse(fragment string) (ImplicitAccessTokenResponse, error) {
	parameters, err := url.ParseQuery(fragment)
	if err != nil {
		return nil, err
	}
	if errs := parameters["error"]; len(errs) > 0 {
		// §4.2.2.1 error
		var errorURI *url.URL
		if errorURIs := parameters["error_uri"]; len(errorURIs) > 0 {
			var err error
			errorURI, err = url.Parse(errorURIs[0])
			if err != nil {
				return nil, errors.Wrap(err, "cannot parse error response: invalid error_uri")
			}
		}
		return ImplicitAccessTokenErrorResponse{
			Error:            errs[0],
			ErrorDescription: parameters.Get("error_description"),
			ErrorURI:         errorURI,
			State:            parameters.Get("state"),
		}, nil
	}
	// §4.2.2 success
	accessTokens := parameters["access_token"]
	if len(accessTokens) == 0 {
		return nil, errors.New("cannot parse response: missing required \"access_token\" parameter")
	}
	tokenTypes := parameters["token_types"]
	if len(tokenTypes) == 0 {
		return nil, errors.New("cannot parse response: missing required \"token_type\" parameter")
	}
	ret := ImplicitAccessTokenSuccessResponse{
		AccessToken: accessTokens[0],
		TokenType:   tokenTypes[0],
		State:       parameters.Get("state"),
	}
	if expiresIns := parameters["expires_in"]; len(expiresIns) != 0 {
		seconds, err := strconv.ParseFloat(expiresIns[0], 64)
		if err != nil {
			return nil, errors.Wrap(err, "cannot parse response: cannot parse \"expires_in\" parameter")
		}
		ret.ExpiresAt = time.Now().Add(time.Duration(seconds * float64(time.Second)))
	}
	if scopes := parameters["scopes"]; len(scopes) != 0 {
		ret.Scope = parseScope(scopes[0])
	}
	return ret, nil
}

// ImplicitAccessTokenResponse encapsulates the possible responses to
// an Authorization Request in the Implicit flow.
//
// This is implemented by ImplicitAccessTokenSuccessResponse and an
// ImplicitAccessTokenErrorResponse.
type ImplicitAccessTokenResponse interface {
	isImplicitAccessTokenResponse()
	GetState() string
}

// An ImplicitAccessTokenSuccessResponse is a successful response to
// an Authorization Request in the Implicit flow, as defined in
// §4.2.2.
type ImplicitAccessTokenSuccessResponse struct {
	AccessToken string    // REQUIRED.
	TokenType   string    // REQUIRED.
	ExpiresAt   time.Time // RECOMMENDED.
	Scope       Scope     // OPTIONAL if identical to scope requested by the client; otherwise REQUIRED.
	State       string    // REQUIRED if the "state" parameter was present in the request.
}

func (r ImplicitAccessTokenSuccessResponse) isImplicitAccessTokenResponse() {}

// GetState returns the state parameter (if any) included in the response.
func (r ImplicitAccessTokenSuccessResponse) GetState() string { return r.State }

// An ImplicitAccessTokenErrorResponse is an error response to an
// Authorization Request in the Implicit flow, as defined in §4.2.2.1.
type ImplicitAccessTokenErrorResponse struct {
	Error            string
	ErrorDescription string
	ErrorURI         *url.URL
	State            string
}

func (r ImplicitAccessTokenErrorResponse) isImplicitAccessTokenResponse() {}

// GetState returns the state parameter (if any) included in the response.
func (r ImplicitAccessTokenErrorResponse) GetState() string { return r.State }

// ErrorMeaning returns a human-readable meaning of the .Error code.
// Returns an emtpy string for unknown error codes.
func (r ImplicitAccessTokenErrorResponse) ErrorMeaning() string {
	ecode := rfc6749.GetImplicitGrantError(r.Error)
	if ecode == nil {
		return ""
	}
	return ecode.Meaning()
}

func newImplicitAccessTokenError(name, meaning string) {
	rfc6749.ExtensionError{
		Name:    name,
		Meaning: meaning,
		UsageLocations: []rfc6749.ErrorUsageLocation{
			rfc6749.ImplicitGrantErrorResponse,
		},
	}.Register()
}

// These are the error codes that may be present in an
// ImplicitAccessTokenErrorResponse, as enumerated in §4.2.2.1.
func init() {
	newImplicitAccessTokenError("invalid_request", ""+
		"The request is missing a required parameter, includes an "+
		"invalid parameter value, includes a parameter more than "+
		"once, or is otherwise malformed.")

	newImplicitAccessTokenError("unauthorized_client", ""+
		"The client is not authorized to request an access token "+
		"using this method.")

	newImplicitAccessTokenError("access_denied", ""+
		"The resource owner or authorization server denied the "+
		"request.")

	newImplicitAccessTokenError("unsupported_response_type", ""+
		"The authorization server does not support obtaining an "+
		"access token using this method.")

	newImplicitAccessTokenError("invalid_scope", ""+
		"The requested scope is invalid, unknown, or malformed.")

	newImplicitAccessTokenError("server_error", ""+
		"The authorization server encountered an unexpected "+
		"condition that prevented it from fulfilling the request.  "+
		"(This error code is needed because a 500 Internal Server "+
		"Error HTTP status code cannot be returned to the client "+
		"via an HTTP redirect.)")

	newImplicitAccessTokenError("temporarily_unavailable", ""+
		"The authorization server is currently unable to handle "+
		"the request due to a temporary overloading or maintenance "+
		"of the server.  (This error code is needed because a 503 "+
		"Service Unavailable HTTP status code cannot be returned "+
		"to the client via an HTTP redirect.)")
}
