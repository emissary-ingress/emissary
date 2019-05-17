// Package rfc6749client implements the "Client" role of the OAuth 2.0 Framework.
package rfc6749client

import (
	"net/http"
	"net/url"
	"strings"

	"github.com/pkg/errors"
)

// validateAuthorizationEndpointURI validates the requirements in §3.1.
func validateAuthorizationEndpointURI(endpoint *url.URL) error {
	if endpoint.Fragment != "" {
		return errors.Errorf("the Authorization Endpoint URI MUST NOT include a fragment component: %v", endpoint)
	}
	return nil
}

// buildAuthorizationRequestURI inserts queryParameters per §3.1.
func buildAuthorizationRequestURI(endpoint *url.URL, queryParameters url.Values) (*url.URL, error) {
	query := endpoint.Query()
	for k, vs := range queryParameters {
		if _, exists := query[k]; exists {
			return nil, errors.Errorf("cannot build Authorization Request URI: cannot insert %q parameter: Authorization Endpoint URI already includes parameter, and request parameters MUST NOT be included more than once", k)
		}
		if len(vs) > 1 {
			return nil, errors.Errorf("cannot build Authorization Request URI: request parameters MUST NOT be included more than once: %q", k)
		}
		query[k] = vs
	}
	ret := *endpoint
	ret.RawQuery = query.Encode()
	return &ret, nil
}

// validateRedirectionEndpointURI validates the requirements in §3.1.2.
func validateRedirectionEndpointURI(endpoint *url.URL) error {
	if !endpoint.IsAbs() {
		return errors.Errorf("the Redirection Endpoint MUST be an absolute URI: %v", endpoint)
	}
	if endpoint.Fragment != "" {
		return errors.Errorf("the Redirection Endpoint URI MUST NOT include a fragment component: %v", endpoint)
	}
	return nil
}

// validateTokenEndpointURI validates the requirements in §3.2.
func validateTokenEndpointURI(endpoint *url.URL) error {
	if endpoint.Fragment != "" {
		return errors.Errorf("the Token Endpoint URI MUST NOT include a fragment component: %v", endpoint)
	}
	return nil
}

// buildTokenURI inserts queryParameters per §3.2.
func buildTokenURI(endpoint *url.URL, queryParameters url.Values) (*url.URL, error) {
	query := endpoint.Query()
	for k, vs := range queryParameters {
		if _, exists := query[k]; exists {
			return nil, errors.Errorf("cannot build Token Request URI: cannot insert %q parameter: Token Endpoint URI already includes parameter, and request parameters MUST NOT be included more than once", k)
		}
		if len(vs) > 1 {
			return nil, errors.Errorf("cannot build Token Request URI: request parameters MUST NOT be included more than once: %q", k)
		}
		query[k] = vs
	}
	ret := *endpoint
	ret.RawQuery = query.Encode()
	return &ret, nil
}

// Scopes represents a list of scopes as defined by §3.3.
type Scopes map[string]struct{}

// String serializes the set of scopes for use as a parameter, per
// §3.3.
func (scopes Scopes) String() string {
	strs := make([]string, 0, len(scopes))
	for k := range scopes {
		strs = append(strs, k)
	}
	return strings.Join(strs, " ")
}

// An AuthorizationCodeClient is Client that utilizes the
// "Authorization Code" Grant-type, as defined by §4.1.
type AuthorizationCodeClient struct {
	clientID              string
	authorizationEndpoint *url.URL
	tokenEndpoint         *url.URL
}

// NewAuthorizationCodeClient creates a new AuthorizationCodeClient as
// defined by §4.1.
func NewAuthorizationCodeClient(
	clientID string,
	authorizationEndpoint *url.URL,
	tokenEndpoint *url.URL,
) (*AuthorizationCodeClient, error) {
	if err := validateAuthorizationEndpointURI(authorizationEndpoint); err != nil {
		return nil, err
	}
	if err := validateAuthorizationEndpointURI(authorizationEndpoint); err != nil {
		return nil, err
	}
	ret := &AuthorizationCodeClient{
		clientID:              clientID,
		authorizationEndpoint: authorizationEndpoint,
		tokenEndpoint:         tokenEndpoint,
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
func (client *AuthorizationCodeClient) AuthorizationRequest(w http.ResponseWriter, r *http.Request, redirectURI *url.URL, scopes Scopes, state string) {
	parameters := url.Values{
		"response_type": {"code"},
		"client_id":     {client.clientID},
	}
	if redirectURI != nil {
		err := validateRedirectionEndpointURI(redirectURI)
		if err != nil {
			err = errors.Wrap(err, "cannot build Authorization Request URI")
			http.Error(w, err.String(), http.StatusInternalServerError)
		}
		parameters.Set("redirect_uri", redirectURI.String())
	}
	if len(scopes) != 0 {
		parameters.Set("scope", Scopes.String())
	}
	if state != "" {
		parameters.Set("state", state)
	}
	requestURI, err := buildAuthorizationRequestURI(client.authorizationEndpoint, parameters)
	if err != nil {
		http.Error(w, err.String(), http.StatusInternalServerError)
	}

	http.Redirect(w, r, requestURI.String(), http.StatusFound)
	return nil
}
