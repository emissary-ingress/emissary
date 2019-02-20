package oauth2handler

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
)

// Rest is a generic rest HTTP client.
type Rest struct {
	AuthorizationEndpoint *url.URL
	TokenEndpoint         *url.URL
	client                *http.Client
	token                 string
}

// NewRestClient creates an instance of a rest client.
func NewRestClient(authorizationEndpoint *url.URL, tokenEndpoint *url.URL) *Rest {
	return &Rest{
		client:                http.DefaultClient,
		AuthorizationEndpoint: authorizationEndpoint,
		TokenEndpoint:         tokenEndpoint,
	}
}

// AuthorizationResponse is used for de-serializing an authorization response.
type AuthorizationResponse struct {
	AccessToken  string `json:"access_token"`
	IDToken      string `json:"id_token"`
	TokenType    string `json:"token_type"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int64  `json:"expires_in"`
}

// AuthorizationRequest structure is used to create authorization request body.
type AuthorizationRequest struct {
	GrantType    string `json:"grant_type"`
	ClientID     string `json:"client_id"`
	Code         string `json:"code"`
	RedirectURL  string `json:"redirect_uri"`
	ClientSecret string `json:"client_secret,omitempty"`
	Audience     string `json:"audience,omitempty"`
}

// Authorize sends a POST request to the IDP.
func (c *Rest) Authorize(a *AuthorizationRequest) (*AuthorizationResponse, error) {
	// build the request
	request, err := http.NewRequest("POST", c.TokenEndpoint.String(), strings.NewReader(url.Values{
		"grant_type":    {a.GrantType},
		"client_id":     {a.ClientID},
		"code":          {a.Code},
		"redirect_uri":  {a.RedirectURL},
		"client_secret": {a.ClientSecret},
		"audience":      {a.Audience},
	}.Encode()))
	if err != nil {
		return nil, err
	}
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	if c.token != "" {
		request.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.token))
	}

	// fire it off
	response, err := c.client.Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	if response.StatusCode >= http.StatusBadRequest {
		body, _ := ioutil.ReadAll(response.Body) // don't let an error here mask the HTTP error
		return nil, fmt.Errorf("HTTP %s: %s", response.Status, string(body))
	}

	// unmarshal the response
	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}
	ret := AuthorizationResponse{}
	if err := json.Unmarshal(body, &ret); err != nil {
		return nil, err
	}

	return &ret, nil
}
