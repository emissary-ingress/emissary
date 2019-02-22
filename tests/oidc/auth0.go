package oidc

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	goquery "github.com/PuerkitoBio/goquery"

	testutil "github.com/datawire/apro/lib/testutil"
)

type auth0 struct {
	*AuthenticationContext
	Audience    string
	ClientID    string
	Tenant      string
	cookieCSRF  string
	cookieAuth0 string
	state       string
}

// auth0UsernameAndPassword login is a struct that is serialized to JSON and sent to the Auth0 authentication endpoint
type auth0UsernameAndPasswordLogin struct {
	Audience     string                 `json:"audience"`  // this might be dynamically discoverable
	ClientID     string                 `json:"client_id"` // this might be dynamically discoverable
	Connection   string                 `json:"connection"`
	CSRF         string                 `json:"_csrf"`     // this value is very important.
	Intstate     string                 `json:"_intstate"` // should always be string literal: "deprecated"
	Password     string                 `json:"password"`
	PopupOptions map[string]interface{} `json:"popup_options"`
	Protocol     string                 `json:"protocol"`
	RedirectURI  string                 `json:"redirect_uri"`
	ResponseType string                 `json:"response_type"`
	Scope        string                 `json:"scope"`
	SSO          bool                   `json:"sso"`
	State        string                 `json:"state"`
	Tenant       string                 `json:"tenant"`
	Username     string                 `json:"username"`
}

func (a *auth0) Authenticate(ctx *AuthenticationContext) (string, error) {
	var token string
	var err error

	assert := testutil.Assert{T: ctx.T}
	a.AuthenticationContext = ctx

	assert.HTTPResponseStatusEQ(a.initialAuthResponse, http.StatusFound)

	// 3. Handle the Redirect and goto the IdP Login URL
	loginUIRedirectURL, err := url.Parse(a.initialAuthResponse.Header.Get("Location"))
	if err != nil {
		return token, err
	}

	// Auth0 hands back relative (path-only) redirects which are useless for subsequent requests. We are talking to the
	// same endpoint as "authRequest" so just use the scheme, host and port info from that.
	loginUIRedirectURL.Scheme = a.initialAuthRequest.URL.Scheme
	loginUIRedirectURL.Host = a.initialAuthRequest.URL.Host
	loginUIRequest, err := createHTTPRequest("GET", *loginUIRedirectURL)
	if err != nil {
		return token, err
	}

	loginUIResponse, err := a.HTTP.Do(loginUIRequest)
	if err != nil {
		return token, err
	}

	assert.HTTPResponseStatusEQ(loginUIResponse, http.StatusOK)

	loginForm := loginUIResponse

	a.state = loginUIRedirectURL.Query().Get("state")
	loginRequest, err := a.createLoginRequest(loginForm)
	if err != nil {
		return token, err
	}

	loginRequest.Header.Add(
		"User-Agent",
		"Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/67.0.3396.62 Safari/537.36")

	loginResponse, err := a.HTTP.Do(loginRequest)
	if err != nil {
		return token, err
	}

	assert.HTTPResponseStatusEQ(loginResponse, 200)

	htmlDoc, err := goquery.NewDocumentFromReader(loginResponse.Body)
	if err != nil {
		return token, err
	}

	form := htmlDoc.Find("form[name=hiddenform]")
	loginCallbackURL, err := url.Parse(form.AttrOr("action", ""))
	if err != nil {
		return token, err
	}

	loginCallbackToken := htmlDoc.Find("input[name=wresult]").AttrOr("value", "")
	loginCallbackCtx := htmlDoc.Find("input[name=wctx]").AttrOr("value", "")

	formData := url.Values{}
	formData.Add("wresult", loginCallbackToken)
	formData.Add("wctx", loginCallbackCtx)

	loginCallbackRequest, err := http.NewRequest("POST", loginCallbackURL.String(), strings.NewReader(formData.Encode()))
	if err != nil {
		return token, err
	}

	// this is all stuff the browser sends so add it as well in case its needed
	SetHeaders(loginCallbackRequest, map[string]string{
		"accept":       "*/*",
		"content-type": "application/x-www-form-urlencoded",
		"cookie":       fmt.Sprintf("_csrf=%s", a.cookieCSRF),
	})

	loginCallbackResponse, err := a.HTTP.Do(loginCallbackRequest)
	if err != nil {
		return token, err
	}

	// back to our callback...
	redirectURL, err := url.Parse(loginCallbackResponse.Header.Get("Location"))
	if err != nil {
		return token, err
	}

	callbackRequest, err := http.NewRequest("POST", redirectURL.String(), strings.NewReader(formData.Encode()))
	if err != nil {
		return token, err
	}

	SetHeaders(callbackRequest, map[string]string{
		"accept":       "*/*",
		"content-type": "application/x-www-form-urlencoded",
		"cookie":       fmt.Sprintf("_csrf=%s", a.cookieCSRF),
	})

	callbackResponse, err := a.HTTP.Do(callbackRequest)
	if err != nil {
		return token, err
	}

	for _, c := range callbackResponse.Cookies() {
		if c.Name == "access_token" {
			token = c.Value
			break
		}
	}

	return token, err
}

func (a *auth0) createLoginRequest(r *http.Response) (*http.Request, error) {

	// get the cookieCSRF token
	for _, c := range r.Cookies() {
		if c.Name == "_csrf" {
			a.cookieCSRF = c.Value
		}
		if c.Name == "auth0" {
			a.cookieAuth0 = c.Value
		}
	}

	loginEndpoint, err := url.Parse(fmt.Sprintf("https://%s.auth0.com/usernamepassword/login", a.Tenant))
	if err != nil {
		return nil, err
	}

	loginParams := auth0UsernameAndPasswordLogin{
		Audience:     a.Audience,
		ClientID:     a.ClientID,
		CSRF:         a.cookieCSRF,
		Connection:   "Username-Password-Authentication",
		Intstate:     "deprecated",
		Password:     a.Password,
		PopupOptions: make(map[string]interface{}),
		Protocol:     "oauth2",
		RedirectURI:  "https://ambassador.localdev.svc.cluster.local/callback",
		ResponseType: "code",
		Scope:        strings.Join(a.Scopes, " "),
		State:        a.state,
		SSO:          true,
		Tenant:       a.Tenant,
		Username:     a.UsernameOrEmail,
	}

	loginParamsBytes, err := json.MarshalIndent(&loginParams, "", "   ")
	if err != nil {
		return nil, err
	}

	loginParamsString := string(loginParamsBytes)
	request, err := http.NewRequest("POST", loginEndpoint.String(), strings.NewReader(loginParamsString))
	if err != nil {
		return nil, err
	}

	// this is all stuff the browser sends so add it as well in case its needed
	SetHeaders(request, map[string]string{
		"accept":       "*/*",
		"content-type": "application/json",
		"cookie":       fmt.Sprintf("_csrf=%s", a.cookieCSRF),
	})

	return request, nil
}
