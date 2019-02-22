package oidc

import (
	"fmt"
	"net/http"
	"net/url"
	"testing"
	"time"

	testutil "github.com/datawire/apro/lib/testutil"
)

func TestAuth0(t *testing.T) {
	httpClient := newHTTPClient(5*time.Second, true)

	ctx := &AuthenticationContext{
		T: t,
		Authenticator: &auth0{
			Audience: "https://ambassador-oauth-e2e.auth0.com/api/v2/",
			ClientID: "DOzF9q7U2OrvB7QniW9ikczS1onJgyiC",
			Tenant:   "ambassador-oauth-e2e",
		},
		HTTP: &httpClient,
		ProtectedResource: url.URL{
			Scheme: "https",
			Host:   "ambassador.localdev.svc.cluster.local",
			Path:   "/auth0/httpbin/headers",
		},
		UsernameOrEmail: "testuser@datawire.com",
		Password:        "TestUser321",
		Scopes:          []string{"openid", "profile", "email"},
	}

	testIDP(ctx)
}

func TestKeycloak(t *testing.T) {
	httpClient := newHTTPClient(5*time.Second, false)

	ctx := &AuthenticationContext{
		T:             t,
		Authenticator: &keycloak{},
		HTTP:          &httpClient,
		ProtectedResource: url.URL{
			Scheme: "https",
			Host:   "ambassador.localdev.svc.cluster.local",
			Path:   "/keycloak/httpbin/headers",
		},
		UsernameOrEmail: "developer",
		Password:        "developer",
	}

	testIDP(ctx)
}

// testIDP performs the full standard authentication flow.
func testIDP(ctx *AuthenticationContext) {
	assert := testutil.Assert{T: ctx.T}

	authenticator, ok := ctx.Authenticator.(AuthenticationDriver)
	if !ok {
		ctx.T.Fatal("Authenticator interface is not implemented")
	}

	// 1. Initiate an HTTP GET request to our protected service
	unauthorizedRequest, err := createHTTPRequest("GET", ctx.ProtectedResource)
	assert.NotError(err)

	unauthorizedResponse, err := ctx.HTTP.Do(unauthorizedRequest)
	assert.NotError(err)
	assert.HTTPResponseStatusEQ(unauthorizedResponse, http.StatusSeeOther)

	// 2. Construct a redirect to the Identity Providers Authorization endpoint. Since we do not have an access token
	// at this point we will end up being redirected.
	redirectURL, err := url.Parse(unauthorizedResponse.Header.Get("Location"))
	assert.NotError(err)

	authRequest, err := createHTTPRequest("GET", *redirectURL)
	assert.NotError(err)

	authResponse, err := ctx.HTTP.Do(authRequest)
	assert.NotError(err)

	// We have been given a response from the authorization endpoint. This is IDP specific and could be the actual
	// login form OR just a redirect. At this point hand the code over to the IDP test implementation and let it
	// drive the authentication process.
	ctx.initialAuthRequest = authRequest
	ctx.initialAuthResponse = authResponse

	accessToken, err := authenticator.Authenticate(ctx)
	assert.NotError(err)
	assert.StrNotEmpty(accessToken)

	// Almost home baby!
	ctx.HTTP.Jar = nil

	unauthorizedRequest.Header.Add("Authorization", fmt.Sprintf("Bearer %s", accessToken))

	// Create a new HTTP client that is entirely unused. This ensures the CookieJar is not re-used. A problem that was
	// discovered during development is that auth0 sets the "access_token" cookie which means the "Authorization"
	// header was being ignored.
	virginHTTPClient := newHTTPClient(5*time.Second, false)

	finalDestination, err := virginHTTPClient.Do(unauthorizedRequest)
	assert.NotError(err)
	assert.HTTPResponseStatusEQ(finalDestination, http.StatusOK)
}
