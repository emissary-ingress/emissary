package rfc6749

import (
	"net/url"

	"github.com/pkg/errors"
)

type accessTokenData struct {
	AccessToken  string
	TokenType    string   
	ExpiresAt    time.Time
	RefreshToken *string
	Scope        Scope 
}

type explicitClientSessionData interface {
	currentAccessToken() *accessTokenData
	setDirty()
	IsDirty() bool
}

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
func (client *explicitClient) refreshToken(session explicitClientSessionData, scope Scope) error {
	accessTokenData := session.currentAccessTokenData()
	if accessTokenData == nil {
		return errors.New("RefreshToken(): CurrentAccessTokenData is nil")
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

	newAccessTokenData := 
	if tokenResponse.RefreshToken != nil {
		session.currentAccessTokenData().RefreshToken = tokenResponse.RefreshToken
		session.setDirty()
	}
	if tokenResponse.Scope != nil {
		session.currentAccessTokenData().Scope = tokenResponse.Scope
	}

	session.setDirty()

	return res, nil
}

func (
