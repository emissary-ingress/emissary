package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
)

// Rest is a generic rest HTTP client.
type Rest struct {
	BaseURL *url.URL
	client  *http.Client
	token   string
}

// NewRestClient creates an instance of a rest client.
func NewRestClient(u *url.URL) *Rest {
	return &Rest{
		client:  http.DefaultClient,
		BaseURL: u,
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

// SetBearerToken allows setting a persistent access token to the the client.
func (c *Rest) SetBearerToken(t string) {
	c.token = t
}

// Authorize sends a POST request to the IDP.
func (c *Rest) Authorize(a *AuthorizationRequest) (*AuthorizationResponse, error) {
	var rq *http.Request
	var err error

	rq, err = c.request("POST", "/oauth/token", *a)
	if err != nil {
		return nil, err
	}

	rs := &AuthorizationResponse{}
	if err := c.do(rq, rs); err != nil {
		return nil, err
	}

	return rs, nil
}

func (c *Rest) request(method, path string, body interface{}) (*http.Request, error) {
	rpath := &url.URL{Path: path}
	url := c.BaseURL.ResolveReference(rpath)

	var buf io.ReadWriter
	if body != nil {
		buf = new(bytes.Buffer)
		err := json.NewEncoder(buf).Encode(body)
		if err != nil {
			return nil, err
		}
	}

	rq, err := http.NewRequest(method, url.String(), buf)
	if err != nil {
		return nil, err
	}

	rq.Header.Set("Accept", "application/json")
	if body != nil {
		rq.Header.Set("Content-Type", "application/json")
	}

	if c.token != "" {
		rq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.token))
	}

	return rq, nil
}

func (c *Rest) do(rq *http.Request, v interface{}) error {
	rs, err := c.client.Do(rq)
	if err != nil {
		return err
	}
	defer rs.Body.Close()

	body, err := ioutil.ReadAll(rs.Body)
	if err != nil {
		return err
	}

	if rs.StatusCode >= http.StatusBadRequest {
		return fmt.Errorf("HTTP %s: %s", rs.Status, string(body))
	}

	return json.Unmarshal(body, v)
}
