package rfc6749client

import (
	"net/http"
	"net/url"
	"strings"

	"github.com/pkg/errors"
)

// A ClientCredentialsClient is a Client that utilizes the "Client
// Credentials" Grant-type, as defined by §4.4.
type ClientCredentialsClient struct {
	tokenEndpoint        *url.URL
	clientAuthentication ClientAuthenticationMethod
}

// NewClientCredentialsClient creates a new ClientCredentialsClient as
// defined by §4.4.
func NewClientCredentialsClient(
	tokenEndpoint *url.URL,
	clientAuthentication ClientAuthenticationMethod,
) (*ClientCredentialsClient, error) {
	if err := validateTokenEndpointURI(tokenEndpoint); err != nil {
		return nil, err
	}
	if clientAuthentication == nil {
		return nil, errors.New("clientAuthentication must be set")
	}
	ret := &ClientCredentialsClient{
		tokenEndpoint:        tokenEndpoint,
		clientAuthentication: clientAuthentication,
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
func (client *ClientCredentialsClient) AccessToken(httpClient *http.Client, scope Scope) (TokenResponse, error) {
	if httpClient == nil {
		httpClient = http.DefaultClient
	}

	parameters := url.Values{
		"grant_type": {"client_credentials"},
	}
	if len(scope) != 0 {
		parameters.Set("scope", scope.String())
	}

	req, err := http.NewRequest("POST", client.tokenEndpoint.String(), strings.NewReader(parameters.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	client.clientAuthentication(req)

	res, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	return parseTokenResponse(res)
}
