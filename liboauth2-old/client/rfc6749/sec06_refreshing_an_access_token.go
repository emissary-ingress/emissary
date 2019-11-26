package rfc6749

import (
	"net/url"
)

// refresh talks to the Authorization Server to exchange a Refresh Token for an Access Token (and
// maybe a new Refresh Token); per §6.
//
// If the authorization flow for the session has not yet been completed, ErrNoAccessToken is
// returned.  If the Authorization Server declined to issue a Refresh Roken during the authorization
// flow, ErrNoRefreshToken is returned.  If the Authorization Server sent a semantically valid error
// response, an error of type TokenErrorResponse is returned.  On protocol errors, an error of a
// different type is returned.
//
// This method is unexported, and accepts an interface, so that the implementation can be shared.
// An exported wrapper around it for each client type takes a concrete type instead of an interface.
// The per-client wrappers live in `sec06_refreshing_an_access_token@${TYPE}.go`.
func (client *explicitClient) refresh(session clientSessionData, scope Scope) error {
	if session.currentAccessToken() == nil {
		return ErrNoAccessToken
	}
	if session.currentAccessToken().RefreshToken == nil {
		return ErrNoRefreshToken
	}

	parameters := url.Values{
		"grant_type":    {"refresh_token"},
		"refresh_token": {*session.currentAccessToken().RefreshToken},
	}
	if len(scope) != 0 {
		parameters.Set("scope", scope.String())
	}

	tokenResponse, err := client.postForm(parameters)
	if err != nil {
		return err
	}

	newAccessTokenData := accessTokenData(tokenResponse)
	if newAccessTokenData.RefreshToken == nil {
		newAccessTokenData.RefreshToken = session.currentAccessToken().RefreshToken
	}
	if len(newAccessTokenData.Scope) == 0 {
		newAccessTokenData.Scope = session.currentAccessToken().Scope
	}

	*session.currentAccessToken() = newAccessTokenData
	session.setDirty()

	return nil
}
