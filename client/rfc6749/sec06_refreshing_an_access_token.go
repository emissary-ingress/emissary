package rfc6749

import (
	"net/url"
)

// refreshToken talks to the Authorization Server to exchange a
// Refresh Token for an Access Token (and maybe a new Refresh Token);
// per §6.
//
// If the authorization flow for the session has not yet been
// completed, ErrNoAccessToken is returned.  If the authorization
// server declined to issue a Refresh Roken during the authorization
// flow, ErrNoRefreshToken is returned.  If the authorization server
// sent a semantically valid error response, an error of type
// TokenErrorResponse is returned.  On protocol errors, an error of a
// different type is returned.
//
// This method is unexported, and accepts an interface, so that the
// implementation can be shared.  An exported wrapper around it for
// each client type takes a concrete type instead of an interface.
func (client *explicitClient) refreshToken(session explicitClientSessionData, scope Scope) error {
	if session.currentAccessTokenData() == nil {
		return ErrNoAccessToken
	}
	if session.currentAccessTokenData().RefreshToken == nil {
		return ErrNoRefreshToken
	}

	parameters := url.Values{
		"grant_type":    {"refresh_token"},
		"refresh_token": {*oldtoken.RefreshToken},
	}
	if len(scope) != 0 {
		parameters.Set("scope", scope.String())
	}

	tokenResponse, err := client.postForm(parameters)
	if err != nil {
		return err
	}

	newAccessTokenData := accessTokenData{
		AccessToken:  tokenResponse.AccessToken,
		TokenType:    tokenResponse.TokenType,
		ExpiresAt:    tokenResponse.ExpiresAt,
		RefreshToken: tokenResponse.RefreshToken,
		Scope:        Scope,
	}
	if newAccessTokenData.RefreshToken == nil {
		newAccessTokenData.RefreshToken = session.currentAccessTokenData().RefreshToken
	}
	if len(newAccessTokenData) == 0 {
		newAccessTokenData.Scope = session.currentAccessTokenData().Scope
	}

	*session.currentAccessTokenData() = newAccessTokenData
	session.setDirty()

	return nil
}

// Yes, the following is gross.  It's the cost of having clear godocs
// with concrete types in the signatures.

// RefreshToken talks to the Authorization Server to exchange a
// Refresh Token for an Access Token (and maybe a new Refresh Token);
// per §6.
//
// If the authorization flow for the session has not yet been
// completed, ErrNoAccessToken is returned.  If the authorization
// server declined to issue a Refresh Roken during the authorization
// flow, ErrNoRefreshToken is returned.  If the authorization server
// sent a semantically valid error response, an error of type
// TokenErrorResponse is returned.  On protocol errors, an error of a
// different type is returned.
func (client *AuthorizationCodeClient) RefreshToken(session *AuthorizationCodeClientSessionData, scope Scope) error {
	return client.refreshToken(session, scope)
}

// RefreshToken talks to the Authorization Server to exchange a
// Refresh Token for an Access Token (and maybe a new Refresh Token);
// per §6.
//
// If the authorization flow for the session has not yet been
// completed, ErrNoAccessToken is returned.  If the authorization
// server declined to issue a Refresh Roken during the authorization
// flow, ErrNoRefreshToken is returned.  If the authorization server
// sent a semantically valid error response, an error of type
// TokenErrorResponse is returned.  On protocol errors, an error of a
// different type is returned.
func (client *ResourceOwnerPasswordCredentialsClient) RefreshToken(session *ResourceOwnerPasswordCredentialsClientSessionData, scope Scope) error {
	return client.refreshToken(session, scope)
}

// RefreshToken talks to the Authorization Server to exchange a
// Refresh Token for an Access Token (and maybe a new Refresh Token);
// per §6.
//
// If the authorization flow for the session has not yet been
// completed, ErrNoAccessToken is returned.  If the authorization
// server declined to issue a Refresh Roken during the authorization
// flow, ErrNoRefreshToken is returned.  If the authorization server
// sent a semantically valid error response, an error of type
// TokenErrorResponse is returned.  On protocol errors, an error of a
// different type is returned.
func (client *ClientCredentialsClient) RefreshToken(session *ClientCredentialsClientSessionData, scope Scope) error {
	return client.refreshToken(session, scope)
}
