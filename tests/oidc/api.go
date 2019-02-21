package oidc

import (
	"net/http"
	"net/url"
	"testing"
)

type AuthenticationContext struct {
	*testing.T

	Authenticator interface{}

	// HTTP is an HTTP client that is configured for use in the IDP Test
	HTTP *http.Client

	// ProtectedResource is the HTTP endpoint that is being accessed.
	ProtectedResource url.URL

	// UsernameOrEmail is a known identity in the provider.
	UsernameOrEmail string

	// Password is the secret credential used to authenticate an identity with the provider
	Password string

	// An array of scope names
	Scopes []string

	// initialAuthRequest is the first request to the IDP authentication endpoint. This may seem useless but it is
	// actually quite necessary if the IDP returns relative redirects as the scheme, host and port will need to be
	// resolved against the URL field on this struct.
	initialAuthRequest *http.Request

	// initialAuthResponse is the response to the first request to the IDP authentication endpoint. Depending on the
	// IDP this may be a login form OR a redirect to some other document that needs to be retrieved.
	initialAuthResponse *http.Response
}

// AuthenticationDriver is an interface that needs to be implemented for each Identity Provider. The driver is
// responsible for performing the steps necessary to login through an Identity Provider's login form.
type AuthenticationDriver interface {

	// Authenticate is called to perform the "login" phase of the OpenID connect handshake
	Authenticate(ctx *AuthenticationContext) (token string, err error)
}
