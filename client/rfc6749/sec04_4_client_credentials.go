package rfc6749

import (
	"net/http"
	"net/url"

	"github.com/pkg/errors"
)

// A ClientCredentialsClient is a Client that utilizes the "Client Credentials" Grant-type, as
// defined by §4.4.
type ClientCredentialsClient struct {
	explicitClient
}

// NewClientCredentialsClient creates a new ClientCredentialsClient as defined by §4.4.
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

// ClientCredentialsClientSessionData is the session data that must be persisted between requests
// when using an ClientCredentialsClient
type ClientCredentialsClientSessionData struct {
	CurrentAccessToken *accessTokenData
	isDirty            bool
}

func (session ClientCredentialsClientSessionData) accessToken() *accessTokenData {
	return session.CurrentAccessToken
}
func (session ClientCredentialsClientSessionData) setDirty() { session.isDirty = true }

// IsDirty indicates whether the session data has been mutated since that last time that it was
// unmarshaled.  This is only useful if you marshal it to and unmarshal it from an external
// datastore.
func (session ClientCredentialsClientSessionData) IsDirty() bool { return session.isDirty }

// AccessToken talks to the Authorization Server to exchange Client credentials for an Access Token
// (and maybe a Refresh Token); submitting the request per §4.4.2, and handling the response per
// §4.4.3.
//
// The scopes argument is optional.
//
// The returned response is either a TokenSuccessResponse or a TokenErrorResponse.
func (client *ClientCredentialsClient) AccessToken(scope Scope) (*ClientCredentialsClientSessionData, error) {
	parameters := url.Values{
		"grant_type": {"client_credentials"},
	}
	if len(scope) != 0 {
		parameters.Set("scope", scope.String())
	}

	tokenResponse, err := client.explicitClient.postForm(parameters)
	if err != nil {
		return nil, err
	}

	if len(tokenResponse.Scope) == 0 {
		tokenResponse.Scope = scope
	}
	return &ClientCredentialsClientSessionData{
		CurrentAccessToken: &tokenResponse,
		isDirty:            true,
	}, nil
}
