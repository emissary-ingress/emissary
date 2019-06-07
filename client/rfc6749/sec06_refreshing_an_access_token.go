package rfc6749

import (
	"net/url"

	"github.com/pkg/errors"
)

// RefreshToken talks to the Authorization Server to exchange a
// Refresh Token for an Access Token (and maybe a new Refresh Token);
// per §6.
//
// oldtoken.RefreshToken must not be nil.
//
// The scope argument is optional, and may be used to obtain a token
// with _reduced_ scope.  It is not valid to list a scope that is not
// present in the original Token.
//
// If the server sent a semantically valid error response, the
// returned error is of type TokenErrorResponse.  On protocol errors,
// a different error type is returned.
func (client *explicitClient) RefreshToken(oldtoken TokenResponse, scope Scope) (TokenResponse, error) {
	if oldtoken.RefreshToken == nil {
		return TokenResponse{}, errors.New("RefreshToken(): oldtoken.RefreshToken must not be nil")
	}

	parameters := url.Values{
		"grant_type":    {"refresh_token"},
		"refresh_token": {*oldtoken.RefreshToken},
	}
	if len(scope) != 0 {
		parameters.Set("scope", scope.String())
	}

	res, err := client.postForm(parameters)
	if err != nil {
		return TokenResponse{}, err
	}

	if res.RefreshToken == nil && oldtoken.RefreshToken != nil {
		res.RefreshToken = oldtoken.RefreshToken
	}
	if res.Scope == nil && oldtoken.Scope != nil {
		res.Scope = oldtoken.Scope
	}

	return res, nil
}
