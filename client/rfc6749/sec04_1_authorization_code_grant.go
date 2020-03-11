package rfc6749

import (
	"fmt"
	"net/http"
	"net/url"

	"github.com/pkg/errors"
)

// An AuthorizationCodeClient is a Client that utilizes the "Authorization Code" Grant-type, as
// defined by §4.1.
//
// There are 3 methods that need to be called during an AuthorizationCodeClient's authorization
// flow:
//
//     client := NewAuthorizationCodeClient(...)
//
//     // 1
//     request, session, err := client.AuthorizationRequest(...)
//     errcheck(err)
//     directUserAgentToMakeRequest(request)
//
//     // 2
//     response := getResponseFromUserAgent()
//     authorizationCode, err := client.ParseAuthorizationResponse(session, response)
//     errcheck(err)
//
//     // 3
//     err := client.AccessToken(session, authorizationCode)
//     errcheck(err)
//
// It is up to Client application implementations to determine the appropriate values for "..." and
// to provide appropriate implementations of "errcheck", "directUserAgentToMakeRequest", and
// "getResponseFromUserAgent".
type AuthorizationCodeClient struct {
	clientID              string
	authorizationEndpoint *url.URL

	explicitClient
	extensionRegistry
}

// NewAuthorizationCodeClient creates a new AuthorizationCodeClient as defined by §4.1.
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

// AuthorizationCodeClientSessionData is the session data that must be persisted between requests
// when using an AuthorizationCodeClient.
type AuthorizationCodeClientSessionData struct {
	Request struct {
		RedirectURI *url.URL
		Scope       Scope
		State       string
	}
	CurrentAccessToken *accessTokenData
	isDirty            bool
}

func (session AuthorizationCodeClientSessionData) currentAccessToken() *accessTokenData {
	return session.CurrentAccessToken
}
func (session *AuthorizationCodeClientSessionData) setDirty() { session.isDirty = true }

// IsDirty indicates whether the session data has been mutated since that last time that it was
// unmarshaled.  This is only useful if you marshal it to and unmarshal it from an external
// datastore.
func (session AuthorizationCodeClientSessionData) IsDirty() bool { return session.isDirty }

// AuthorizationRequest returns an URI that the Client should direct the User-Agent to perform a GET
// request for, in order to perform an Authorization Request, per §4.1.1.
//
// OAuth arguments:
//
//  - redirectURI: OPTIONAL if exactly 1 complete Redirection Endpoint was registered with the
//    Authorization Server when registering the Client.  If the Client was not registered with the
//    Authorization Server, it was registered with 0 Redirection Endpoints, it was registered with a
//    partial Redirection Endpoint, or it was registered with more than 1 Redirection Endpoint, then
//    this argument is REQUIRED.
//
//  - scope: OPTIONAL.
//
//  - state: RECOMMENDED.
//
//  - extraParams: Extra parameters used by OAuth extensions.  It is not valid to specify a
//    parameter used by OAuth itself ("response_type", "client_id", "redirect_uri", "scope", or
//    "state"); if you do specify one of these parameters, it will return an error.
//
// The Client is free to use whichever redirection mechanisms it has available to it (perhaps a
// plain HTTP redirect, or perhaps something fancy with JavaScript).  Note that if using an HTTP
// redirect, that 302 "Found" may or MAY NOT convert POST->GET; and that to reliably have the
// User-Agent perform a GET, one should use 303 "See Other" which MUST convert to GET.
func (client *AuthorizationCodeClient) AuthorizationRequest(
	redirectURI *url.URL,
	scope Scope,
	state string,
	extraParams map[string]string,
) (*url.URL, *AuthorizationCodeClientSessionData, error) {
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
	for k, v := range extraParams {
		_, conflict := map[string]struct{}{
			"response_type": {},
			"client_id":     {},
			"redirect_uri":  {},
			"scope":         {},
			"state":         {},
		}[k]
		if conflict {
			return nil, nil, errors.Errorf("may not manually specify built-in parameter %q", k)
		}
		parameters.Set(k, v)
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

// ParseAuthorizationResponse parses the Authorization Response out from the HTTP request URL, as
// specified by §4.1.2.
//
// This should be called from the http.Handler for the Client's Redirection Endpoint.
//
// If the server sent a semantically valid error response, the returned error is of type
// AuthorizationCodeGrantErrorResponse.  On protocol errors, a different error type is returned.
func (client *AuthorizationCodeClient) ParseAuthorizationResponse(
	session *AuthorizationCodeClientSessionData,
	requestURL *url.URL,
) (authorizationCode string, err error) {
	parameters := requestURL.Query()

	// The "state" parameter is shared by both success and error responses.  Let's check this
	// early, to avoid unnecessary resource usage.
	if parameters.Get("state") != session.Request.State {
		return "", errors.WithStack(XSRFError("refusing to parse response: response state parameter does not match request state parameter; XSRF attack likely"))
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

// An AuthorizationCodeGrantErrorResponse is an error response to an Authorization Request in the
// Authorization Code flow, as defined in §4.1.2.1.
type AuthorizationCodeGrantErrorResponse struct {
	// REQUIRED.  A single ASCII error code.
	ErrorCode string

	// OPTIONAL.  Human-readable ASCII providing additional information, used to assist the
	// client developer.
	ErrorDescription string

	// OPTIONAL.  A URI identifying a human-readable web page with information about the error,
	// used to provide the client developer with additional information.
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

// These are the built-in error codes that may be present in an
// AuthorizationCodeAuthorizationErrorResponse, as enumerated in §4.1.2.1.  This set may be extended
// by extensions error registry.
func newBuiltInAuthorizationCodeGrantErrors() map[string]ExtensionError {
	ret := make(map[string]ExtensionError)
	add := func(name, meaning string) {
		if _, set := ret[name]; set {
			panic(errors.Errorf("authorization code grant error=%q already registered", name))
		}
		ret[name] = ExtensionError{
			Name:                   name,
			UsageLocations:         []ErrorUsageLocation{LocationAuthorizationCodeGrantErrorResponse},
			RelatedExtension:       "(built-in)",
			ChangeController:       "IETF",
			SpecificationDocuments: []string{"RFC 6749"},

			Meaning: meaning,
		}
	}

	add("invalid_request", ""+
		"The request is missing a required parameter, includes an "+
		"invalid parameter value, includes a parameter more than "+
		"once, or is otherwise malformed.")

	add("unauthorized_client", ""+
		"The client is not authorized to request an authorization "+
		"code using this method.")

	add("access_denied", ""+
		"The resource owner or authorization server denied the "+
		"request.")

	add("unsupported_response_type", ""+
		"The authorization server does not support obtaining an "+
		"authorization code using this method.")

	add("invalid_scope", ""+
		"The requested scope is invalid, unknown, or malformed.")

	add("server_error", ""+
		"The authorization server encountered an unexpected "+
		"condition that prevented it from fulfilling the request.  "+
		"(This error code is needed because a 500 Internal Server "+
		"Error HTTP status code cannot be returned to the client "+
		"via an HTTP redirect.)")

	add("temporarily_unavailable", ""+
		"The authorization server is currently unable to handle "+
		"the request due to a temporary overloading or maintenance "+
		"of the server.  (This error code is needed because a 503 "+
		"Service Unavailable HTTP status code cannot be returned "+
		"to the client via an HTTP redirect.)")

	return ret
}

// AccessToken talks to the Authorization Server to exchange an Authorization Code (obtained from
// `.ParseAuthorizationResponse()`) for an Access Token (and maybe a Refresh Token); submitting the
// request per §4.1.3, and handling the response per §4.1.4.
//
// If the call is successful, the Access Token information is stored in to the session data, and the
// session data may then be used with `.AuthorizationForResourceRequest()` and (if a Refresh Token
// was included) `.Refresh()`.
//
// If the Authorization Server sent a semantically valid error response, an error of type
// TokenErrorResponse is returned.  On protocol errors, an error of a different type is returned.
func (client *AuthorizationCodeClient) AccessToken(
	session *AuthorizationCodeClientSessionData,
	authorizationCode string,
) error {
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

	tokenResponse, err := client.postForm(parameters)
	if err != nil {
		return err
	}

	newAccessTokenData := accessTokenData(tokenResponse)
	if len(newAccessTokenData.Scope) == 0 {
		newAccessTokenData.Scope = session.Request.Scope
	}

	session.CurrentAccessToken = &newAccessTokenData
	session.isDirty = true
	return nil
}
