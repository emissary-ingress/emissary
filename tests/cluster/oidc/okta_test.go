// +build test

package oidc

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/Jeffail/gabs"
	"github.com/PuerkitoBio/goquery"
	"github.com/datawire/apro/lib/oidctest"
	"github.com/datawire/apro/lib/testutil"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"
)

func TestOkta(t *testing.T) {
	httpClient := oidctest.NewHTTPClient(5*time.Second, true)

	ctx := &oidctest.AuthenticationContext{
		T: t,
		Authenticator: &okta{
			Audience: "api://default",
			ClientID: "0oaeshpr0wKNbyWQn356",
			Tenant:   "dev-264701.okta.com",
		},
		HTTP: &httpClient,
		ProtectedResource: url.URL{
			Scheme: "https",
			Host:   "ambassador.standalone.svc.cluster.local",
			Path:   "/okta/httpbin/headers",
		},
		UsernameOrEmail: "testificate+000@datawire.io",
		Password:        "Qwerty123",
	}

	oidctest.TestIDP(ctx)
}

type okta struct {
	*oidctest.AuthenticationContext
	Audience     string
	ClientID     string
	Tenant       string
	SessionToken string
	FromURI      string
}

func (o *okta) Authenticate(ctx *oidctest.AuthenticationContext) (string, error) {
	var token string
	var err error

	assert := testutil.Assert{T: ctx.T}
	o.AuthenticationContext = ctx

	assert.HTTPResponseStatusEQ(o.InitialAuthResponse, http.StatusFound)

	// 3. Handle the Redirect and goto the IdP Login URL
	loginUIRedirectURL, err := url.Parse(o.InitialAuthResponse.Header.Get("Location"))
	if err != nil {
		return token, err
	}

	loginUIRequest, err := oidctest.CreateHTTPRequest("GET", *loginUIRedirectURL)
	if err != nil {
		return token, err
	}

	loginUIResponse, err := o.HTTP.Do(loginUIRequest)
	if err != nil {
		return token, err
	}

	assert.HTTPResponseStatusEQ(loginUIResponse, http.StatusOK)

	loginForm := loginUIResponse
	htmlDoc, err := goquery.NewDocumentFromReader(loginForm.Body)
	if err != nil {
		return token, err
	}

	// the "fromURI" hidden form input contains an input token that is used for CSRF protection reasons.
	fromURIHiddenInput := htmlDoc.Find("input[id=fromURI]")
	o.FromURI = fromURIHiddenInput.AttrOr("value", "")
	if o.FromURI == "" {
		return token, errors.New("failed to retrieve fromURI")
	}

	loginRequest, err := o.createLoginRequest(loginForm)
	if err != nil {
		return token, err
	}

	loginResponse, err := o.HTTP.Do(loginRequest)
	if err != nil {
		return token, err
	}

	assert.HTTPResponseStatusEQ(loginResponse, 200)

	// grab the sessionToken
	jsonBytes, err := ioutil.ReadAll(loginResponse.Body)
	jsonParsed, err := gabs.ParseJSON(jsonBytes)
	o.SessionToken = jsonParsed.Path("sessionToken").Data().(string)

	sessionCookieRequest, _, err := o.createSessionCookieRequest()
	if err != nil {
		return token, err
	}

	sessionCookieResponse, err := o.HTTP.Do(sessionCookieRequest)
	if err != nil {
		return token, err
	}

	assert.HTTPResponseStatusEQ(sessionCookieResponse, 302)
	sessionCookieRedirectURL, err := url.Parse(sessionCookieResponse.Header.Get("Location"))
	sessionCookieRedirect, err := oidctest.CreateHTTPRequest("GET", *sessionCookieRedirectURL)
	if err != nil {
		return token, err
	}

	sessionCookieRedirectResponse, err := o.HTTP.Do(sessionCookieRedirect)
	if err != nil {
		return token, err
	}
	assert.HTTPResponseStatusEQ(sessionCookieResponse, 302)

	loginCallbackRequestURL, err := url.Parse(sessionCookieRedirectResponse.Header.Get("Location"))
	loginCallbackRequest, err := http.NewRequest("GET", loginCallbackRequestURL.String(), nil)
	if err != nil {
		return token, err
	}

	loginCallbackResponse, err := o.HTTP.Do(loginCallbackRequest)
	if err != nil {
		return token, err
	}

	for _, c := range loginCallbackResponse.Cookies() {
		if c.Name == "access_token" {
			token = c.Value
			break
		}
	}

	return token, err
}

func (o *okta) createSessionCookieRequest() (*http.Request, url.Values, error) {
	var request *http.Request
	var params url.Values
	var err error

	redirectUrl := fmt.Sprintf("https://%s%s", o.Tenant, o.FromURI)
	//fmt.Println(redirectUrl)

	params = url.Values{}
	params.Set("checkAccountSetupComplete", "true")
	params.Set("token", o.SessionToken)
	//params.Set("redirectUrl", redirectUrl)

	request, err = http.NewRequest("GET", fmt.Sprintf("https://%s/login/sessionCookieRedirect?redirectUrl=%s&token=%s", o.Tenant, redirectUrl, o.SessionToken), nil)
	if err != nil {
		return request, params, err
	}

	return request, params, err
}

func (o *okta) createLoginRequest(loginForm *http.Response) (*http.Request, error) {
	var request *http.Request
	var err error

	// {"password":"QWE89pal!","username":"plombardi@datawire.io","options":{"warnBeforePasswordExpired":true,"multiOptionalFactorEnroll":true}}
	loginOptions := struct {
		WarnBeforePasswordExpired bool `json:"warnBeforePasswordExpired"`
		MultiOptionalFactorEnroll bool `json:"multiOptionalFactorEnroll"`
	}{true, true}

	loginParams := struct {
		Username string      `json:"username"`
		Password string      `json:"password"`
		Options  interface{} `json:"options"`
	}{
		o.UsernameOrEmail,
		o.Password,
		loginOptions,
	}

	loginParamsBytes, err := json.MarshalIndent(&loginParams, "", "   ")
	if err != nil {
		return nil, err
	}

	loginParamsString := string(loginParamsBytes)
	request, err = http.NewRequest("POST", fmt.Sprintf("https://%s/api/v1/authn", o.Tenant), strings.NewReader(loginParamsString))
	if err != nil {
		return nil, err
	}

	// this is all stuff the browser sends so add it as well in case its needed
	oidctest.SetHeaders(request, map[string]string{
		"accept":       "*/*",
		"content-type": "application/json",
		"referer":      fmt.Sprintf("https://%s/login/login.htm?fromURI=%s", o.Tenant, o.FromURI),
	})

	return request, err
}
