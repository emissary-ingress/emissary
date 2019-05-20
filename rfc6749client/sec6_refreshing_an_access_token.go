package rfc6749client

import (
	"net/http"
	"net/url"
)

// RefreshToken talks to the Authorization Server to exchange a
// Refresh Token for an Access Token (and maybe a new Refresh Token);
// per §6.
//
// The scope argument is optional, and may be used to obtain a token
// with _reduced_ scope.  It is not valid to list a scope that is not
// present in the original Token.
func (client *explicitClient) RefreshToken(httpClient *http.Client, refreshToken string, scope Scope) (TokenResponse, error) {
	parameters := url.Values{
		"grant_type":    {"refresh_token"},
		"refresh_token": {refreshToken},
	}
	if scope != nil {
		parameters.Set("scope", scope.String())
	}

	return client.postForm(httpClient, parameters)

	// TODO: store the new Refresh Token
}
