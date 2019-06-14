package rfc6749

import (
	"net/http"
	"net/url"
)

// A ResourceOwnerPasswordCredentialsClient is a Client that utilizes the "Resource Owner Password
// Credentials" Grant-type, as defined by §4.3.
type ResourceOwnerPasswordCredentialsClient struct {
	explicitClient
	extensionRegistry
}

// NewResourceOwnerPasswordCredentialsClient creates a new ResourceOwnerPasswordCredentialsClient as
// defined by §4.3.
func NewResourceOwnerPasswordCredentialsClient(
	tokenEndpoint *url.URL,
	clientAuthentication ClientAuthenticationMethod,
	httpClient *http.Client,
) (*ResourceOwnerPasswordCredentialsClient, error) {
	if err := validateTokenEndpointURI(tokenEndpoint); err != nil {
		return nil, err
	}
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	ret := &ResourceOwnerPasswordCredentialsClient{
		explicitClient: explicitClient{
			tokenEndpoint:        tokenEndpoint,
			clientAuthentication: clientAuthentication,
			httpClient:           httpClient,
		},
	}
	return ret, nil
}

// ResourceOwnerPasswordCredentialsClientSessionData is the session data that must be persisted
// between requests when using an ResourceOwnerPasswordCredentialsClient
type ResourceOwnerPasswordCredentialsClientSessionData struct {
	CurrentAccessToken *accessTokenData
	isDirty            bool
}

func (session ResourceOwnerPasswordCredentialsClientSessionData) currentAccessToken() *accessTokenData {
	return session.CurrentAccessToken
}
func (session *ResourceOwnerPasswordCredentialsClientSessionData) setDirty() { session.isDirty = true }

// IsDirty indicates whether the session data has been mutated since that last time that it was
// unmarshaled.  This is only useful if you marshal it to and unmarshal it from an external
// datastore.
func (session ResourceOwnerPasswordCredentialsClientSessionData) IsDirty() bool {
	return session.isDirty
}

// AuthorizationRequest talks to the Authorization Server to exchange a username and password for an
// Access Token (and maybe a Refresh Token); submitting the request per §4.3.2, and handling the
// response per §4.3.3.
//
// The scopes argument is optional.
//
// If the Authorization Server sent a semantically valid error response, an error of type
// TokenErrorResponse is returned. On protocol errors, an error of a different type is returned.
func (client *ResourceOwnerPasswordCredentialsClient) AuthorizationRequest(
	username string,
	password string,
	scope Scope,
) (*ResourceOwnerPasswordCredentialsClientSessionData, error) {
	parameters := url.Values{
		"grant_type": {"password"},
		"username":   {username},
		"password":   {password},
	}
	if len(scope) != 0 {
		parameters.Set("scope", scope.String())
	}

	tokenResponse, err := client.explicitClient.postForm(parameters)
	if err != nil {
		return nil, err
	}

	newAccessTokenData := accessTokenData(tokenResponse)
	if len(newAccessTokenData.Scope) == 0 {
		newAccessTokenData.Scope = scope
	}

	return &ResourceOwnerPasswordCredentialsClientSessionData{
		CurrentAccessToken: &newAccessTokenData,
		isDirty:            true,
	}, nil
}
