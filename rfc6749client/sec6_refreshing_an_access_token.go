package rfc6749client

import (
	"net/http"
	"net/url"
	"strings"
)

// RefreshToken talks to the Authorization Server to exchange a
// Refresh Token for an Access Token (and maybe a new Refresh Token);
// per §6.
//
// The scope argument is optional, and may be used to obtain a token
// with _reduced_ scope.  It is not valid to list a scope that is not
// present in the original Token.
//
// BUG(lukeshu) RefreshToken should not be tied to
// AuthorizationCodeClient.
func (client *AuthorizationCodeClient) RefreshToken(httpClient *http.Client, refreshToken string, scope Scope) (TokenResponse, error) {
	if httpClient == nil {
		httpClient = http.DefaultClient
	}

	parameters := url.Values{
		"grant_type":    {"refresh_token"},
		"refresh_token": {refreshToken},
	}
	if scope != nil {
		parameters.Set("scope", scope.String())
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

	// TODO: store the new Refresh Token
}
