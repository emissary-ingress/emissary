package rfc6749

import (
	"net/http"
	"net/url"
)

// A ResourceOwnerPasswordCredentialsClient is a Client that utilizes
// the "Resource Owner Password Credentials" Grant-type, as defined by
// §4.3.
type ResourceOwnerPasswordCredentialsClient struct {
	explicitClient
}

// NewResourceOwnerPasswordCredentialsClient creates a new
// ResourceOwnerPasswordCredentialsClient as defined by §4.3.
func NewResourceOwnerPasswordCredentialsClient(
	tokenEndpoint *url.URL,
	clientAuthentication ClientAuthenticationMethod,
) (*ResourceOwnerPasswordCredentialsClient, error) {
	if err := validateTokenEndpointURI(tokenEndpoint); err != nil {
		return nil, err
	}
	ret := &ResourceOwnerPasswordCredentialsClient{
		explicitClient: explicitClient{
			tokenEndpoint:        tokenEndpoint,
			clientAuthentication: clientAuthentication,
		},
	}
	return ret, nil
}

// AccessToken talks to the Authorization Server to exchange a
// username and password for an Access Token (and maybe a Refresh
// Token); submitting the request per §4.3.2, and handling the
// response per §4.3.3.
//
// The scopes argument is optional.
//
// The returned response is either a TokenSuccessResponse or a
// TokenErrorResponse.
func (client *ResourceOwnerPasswordCredentialsClient) AccessToken(httpClient *http.Client, username string, password string, scope Scope) (TokenResponse, error) {
	parameters := url.Values{
		"grant_type": {"password"},
		"username":   {username},
		"password":   {password},
	}
	if len(scope) != 0 {
		parameters.Set("scope", scope.String())
	}

	return client.explicitClient.postForm(httpClient, parameters)
}
