package rfc6749

import (
	"fmt"
	"net/url"
	"strconv"
	"time"

	"github.com/pkg/errors"
)

// An ImplicitClient is a Client that utilizes the "Implicit" Grant-type, as defined by §4.2.
type ImplicitClient struct {
	clientID              string
	authorizationEndpoint *url.URL
	extensionRegistry
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

// ImplicitClientSessionData is the session data that must be persisted between requests when using
// an ImplicitClient.
type ImplicitClientSessionData struct {
	Request struct {
		RedirectURI *url.URL
		Scope       Scope
		State       string
	}
	CurrentAccessToken *accessTokenData
	isDirty            bool
}

func (session ImplicitClientSessionData) currentAccessToken() *accessTokenData {
	return session.CurrentAccessToken
}
func (session *ImplicitClientSessionData) setDirty() { session.isDirty = true }

// IsDirty indicates whether the session data has been mutated since that last time that it was
// unmarshaled.  This is only useful if you marshal it to and unmarshal it from an external
// datastore.
func (session ImplicitClientSessionData) IsDirty() bool { return session.isDirty }

// AuthorizationRequest returns an URI that the Client should direct the User-Agent to perform a GET
// request for, in order to perform an Authorization Request, per §4.2.1.
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
// The Client is free to use whichever redirection mechanisms it has available to it (perhaps a
// plain HTTP redirect, or perhaps something fancy with JavaScript).  Note that if using an HTTP
// redirect, that 302 "Found" may or MAY NOT convert POST->GET; and that to reliably have the
// User-Agent perform a GET, one should use 303 "See Other" which MUST convert to GET.
func (client *ImplicitClient) AuthorizationRequest(
	redirectURI *url.URL,
	scope Scope,
	state string,
) (*url.URL, *ImplicitClientSessionData, error) {
	parameters := url.Values{
		"response_type": {"token"},
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

	session := &ImplicitClientSessionData{}
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

// ParseAuthorizationResponse parses the URI fragment that contains the Access Token Response, as
// specified by §4.2.2.
//
// The fragment is normally not accessible to HTTP servers; you will need to use JavaScript in the
// user-agent to somehow get it to the Client HTTP server.
//
// If the Authorization Server sent a semantically valid error response, the returned error is of
// type ImplicitGrantErrorResponse.  On protocol errors, a different error type is returned.
func (client *ImplicitClient) ParseAuthorizationResponse(
	session *ImplicitClientSessionData,
	fragment string,
) error {
	parameters, err := url.ParseQuery(fragment)
	if err != nil {
		return errors.Wrap(err, "cannot parse response")
	}

	// The "state" parameter is shared by both success and error responses.  Let's check this
	// early, to avoid unnecessary resource usage.
	if parameters.Get("state") != session.Request.State {
		return errors.WithStack(XSRFError("refusing to parse response: response state parameter does not match request state parameter; XSRF attack likely"))
	}

	if errs := parameters["error"]; len(errs) > 0 {
		// §4.2.2.1 error
		var errorURI *url.URL
		if errorURIs := parameters["error_uri"]; len(errorURIs) > 0 {
			var err error
			errorURI, err = url.Parse(errorURIs[0])
			if err != nil {
				return errors.Wrap(err, "cannot parse error response: invalid error_uri")
			}
		}
		return ImplicitGrantErrorResponse{
			ErrorCode:        errs[0],
			ErrorDescription: parameters.Get("error_description"),
			ErrorURI:         errorURI,
		}
	}
	// §4.2.2 success
	accessTokens := parameters["access_token"]
	if len(accessTokens) == 0 {
		return errors.New("cannot parse response: missing required \"access_token\" parameter")
	}
	tokenTypes := parameters["token_type"]
	if len(tokenTypes) == 0 {
		return errors.New("cannot parse response: missing required \"token_type\" parameter")
	}
	newAccessTokenData := &accessTokenData{
		AccessToken: accessTokens[0],
		TokenType:   tokenTypes[0],
	}
	if expiresIns := parameters["expires_in"]; len(expiresIns) > 0 {
		seconds, err := strconv.ParseFloat(expiresIns[0], 64)
		if err != nil {
			return errors.Wrap(err, "cannot parse response: cannot parse \"expires_in\" parameter")
		}
		newAccessTokenData.ExpiresAt = time.Now().Add(time.Duration(seconds * float64(time.Second)))
	}
	if scopes := parameters["scopes"]; len(scopes) > 0 {
		newAccessTokenData.Scope = ParseScope(scopes[0])
	} else {
		newAccessTokenData.Scope = session.Request.Scope
	}

	session.CurrentAccessToken = newAccessTokenData
	session.isDirty = true
	return nil
}

// An ImplicitGrantErrorResponse is an error response to an Authorization Request in the Implicit
// flow, as defined in §4.2.2.1.
type ImplicitGrantErrorResponse struct {
	ErrorCode        string
	ErrorDescription string
	ErrorURI         *url.URL
}

func (r ImplicitGrantErrorResponse) Error() string {
	ret := fmt.Sprintf("implicit grant error response: error=%q", r.ErrorCode)
	if r.ErrorDescription != "" {
		ret = fmt.Sprintf("%s error_description=%q", ret, r.ErrorDescription)
	}
	if r.ErrorURI != nil {
		ret = fmt.Sprintf("%s error_uri=%q", ret, r.ErrorURI.String())
	}
	return ret
}

// These are the error codes that may be present in an ImplicitAccessTokenErrorResponse, as
// enumerated in §4.2.2.1.  This set may be extended by the extensions error registry.
func newBuiltInImplicitGrantErrors() map[string]ExtensionError {
	ret := make(map[string]ExtensionError)
	add := func(name, meaning string) {
		if _, set := ret[name]; set {
			panic(errors.Errorf("implicit grant error=%q already registered", name))
		}
		ret[name] = ExtensionError{
			Name:                   name,
			UsageLocations:         []ErrorUsageLocation{LocationImplicitGrantErrorResponse},
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
		"The client is not authorized to request an access token "+
		"using this method.")

	add("access_denied", ""+
		"The resource owner or authorization server denied the "+
		"request.")

	add("unsupported_response_type", ""+
		"The authorization server does not support obtaining an "+
		"access token using this method.")

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
