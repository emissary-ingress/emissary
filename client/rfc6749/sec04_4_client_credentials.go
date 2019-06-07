package rfc6749

import (
	"net/http"
	"net/url"

	"github.com/pkg/errors"
)

// A ClientCredentialsClient is a Client that utilizes the "Client
// Credentials" Grant-type, as defined by §4.4.
type ClientCredentialsClient struct {
	explicitClient
}

// NewClientCredentialsClient creates a new ClientCredentialsClient as
// defined by §4.4.
func NewClientCredentialsClient(
	tokenEndpoint *url.URL,
	clientAuthentication ClientAuthenticationMethod,
	httpClient *http.Client,
) (*ClientCredentialsClient, error) {
	if err := validateTokenEndpointURI(tokenEndpoint); err != nil {
		return nil, err
	}
	if clientAuthentication == nil {
		return nil, errors.New("clientAuthentication must be set")
	}
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	ret := &ClientCredentialsClient{
		explicitClient: explicitClient{
			tokenEndpoint:        tokenEndpoint,
			clientAuthentication: clientAuthentication,
			httpClient:           httpClient,
		},
	}
	return ret, nil
}

// AccessToken talks to the Authorization Server to exchange Client
// credentials for an Access Token (and maybe a Refresh Token);
// submitting the request per §4.4.2, and handling the response per
// §4.4.3.
//
// The scopes argument is optional.
//
// The returned response is either a TokenSuccessResponse or a
// TokenErrorResponse.
func (client *ClientCredentialsClient) AccessToken(scope Scope) (TokenResponse, error) {
	parameters := url.Values{
		"grant_type": {"client_credentials"},
	}
	if len(scope) != 0 {
		parameters.Set("scope", scope.String())
	}

	return client.explicitClient.postForm(parameters)
}
