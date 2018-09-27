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

// Rest ..
type Rest struct {
	BaseURL *url.URL

	client *http.Client
	token  string
}

// NewRestClient ..
func NewRestClient(u *url.URL) *Rest {
	return &Rest{
		client:  http.DefaultClient,
		BaseURL: u,
	}
}

// AuthResponse TODO(gsagula): comment
type AuthResponse struct {
	Token string `json:"access_token"`
	Type  string `json:"token_type"`
}

// AuthorizationResponse used for de-serializing response from /oauth/token.
type AuthorizationResponse struct {
	AccessToken  string `json:"access_token"`
	IDToken      string `json:"id_token"`
	TokenType    string `json:"token_type"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int64  `json:"expires_in"`
}

// AuthorizationRequest ..
type AuthorizationRequest struct {
	GrantType    string `json:"grant_type"`
	ClientID     string `json:"client_id"`
	Code         string `json:"code"`
	RedirectURL  string `json:"redirect_uri"`
	ClientSecret string `json:"client_secret,omitempty"`
	Audience     string `json:"audience,omitempty"`
}

// SetBearerToken ...
// TODO(gsagula): might want also expire and refresh token.
func (c *Rest) SetBearerToken(t string) {
	c.token = t
}

// POSTAuthorization ..
func (c *Rest) POSTAuthorization(a *AuthorizationRequest) (*AuthorizationResponse, error) {
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

// Auth0 client specific API

// Auth0Client ..
type Auth0Client struct {
	rest     *Rest
	secret   string
	clientID string
	audience string
	access   *AuthorizationResponse
}

// NewAuth0Client ..
func NewAuth0Client(rest *Rest, secret string, clientID string, aud string) *Auth0Client {
	return &Auth0Client{
		rest:     rest,
		secret:   secret,
		clientID: clientID,
		audience: aud,
	}
}

// Authorize uses client credentials to fetch the access token.
func (a *Auth0Client) Authorize() error {
	rq := &AuthorizationRequest{
		ClientID:     a.clientID,
		ClientSecret: a.secret,
		Audience:     a.audience,
		GrantType:    "client_credentials",
	}

	rs, err := a.rest.POSTAuthorization(rq)
	if err != nil {
		return err
	}

	a.rest.SetBearerToken(rs.AccessToken)
	return nil
}

// Client ..
type Client struct {
	ClientID   string   `json:"client_id"`
	Callbacks  []string `json:"callbacks"`
	GrantTypes []string `json:"grant_types"`
}

// GetClients ..
func (a *Auth0Client) GetClients() (*[]Client, error) {
	var rq *http.Request
	var err error

	rq, err = a.rest.request("GET", "/api/v2/clients", "")
	if err != nil {
		return nil, err
	}

	rs := &[]Client{}
	if err := a.rest.do(rq, rs); err != nil {
		return nil, err
	}

	return rs, nil
}

// Grant ..
// TODO(gsagula): might be worth checking the scopes as well.
type Grant struct {
	ClientID string `json:"client_id"`
	Audience string `json:"audience"`
}

// GetClientGrants ..
func (a *Auth0Client) GetClientGrants() (*[]Grant, error) {
	var rq *http.Request
	var err error

	rq, err = a.rest.request("GET", "/api/v2/grants", "")
	if err != nil {
		return nil, err
	}

	rs := &[]Grant{}
	if err := a.rest.do(rq, rs); err != nil {
		return nil, err
	}

	return rs, nil
}
