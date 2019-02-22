// +build ignore

package oidc

import (
	"fmt"
	"github.com/datawire/apro/lib/oidctest"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	goquery "github.com/PuerkitoBio/goquery"

	testutil "github.com/datawire/apro/lib/testutil"
)

func TestKeycloak(t *testing.T) {
	httpClient := oidctest.NewHTTPClient(5*time.Second, false)

	ctx := &oidctest.AuthenticationContext{
		T:             t,
		Authenticator: &keycloak{},
		HTTP:          &httpClient,
		ProtectedResource: url.URL{
			Scheme: "https",
			Host:   "ambassador.localdev.svc.cluster.local",
			Path:   "/keycloak/httpbin/headers",
		},
		UsernameOrEmail: "developer",
		Password:        "developer",
	}

	oidctest.TestIDP(ctx)
}

type keycloak struct {
	*oidctest.AuthenticationContext
	cookieAuthSessionID   string
	cookieKeycloakRestart string
}

func (k *keycloak) Authenticate(ctx *oidctest.AuthenticationContext) (string, error) {
	var token string
	var err error

	assert := testutil.Assert{T: ctx.T}
	k.AuthenticationContext = ctx

	assert.HTTPResponseStatusEQ(k.InitialAuthResponse, http.StatusOK)

	loginRequest, loginParams, err := k.createLoginRequest(k.InitialAuthResponse)
	if err != nil {
		return token, err
	}

	loginResponse, err := k.HTTP.Do(loginRequest)
	if err != nil {
		return token, err
	}

	assert.HTTPResponseStatusEQ(loginResponse, http.StatusFound)
	cookies, err := oidctest.ExtractCookies(loginResponse,
		[]string{"KC_RESTART", "KEYCLOAK_IDENTITY", "KEYCLOAK_SESSION", "KEYCLOAK_REMEMBER_ME"})

	if err != nil {
		return token, err
	}

	loginCallbackRequest, err := http.NewRequest("POST", loginResponse.Header.Get("Location"), strings.NewReader(loginParams.Encode()))
	if err != nil {
		return token, err
	}

	oidctest.SetHeaders(loginCallbackRequest, map[string]string{
		"accept":       "*/*",
		"content-type": "application/x-www-form-urlencoded",
		"cookie":       oidctest.FormatCookieHeaderFromCookieMap(cookies),
	})

	loginCallbackResponse, err := k.HTTP.Do(loginCallbackRequest)
	if err != nil {
		return token, err
	}

	assert.HTTPResponseStatusEQ(loginCallbackResponse, http.StatusTemporaryRedirect)
	cookies, err = oidctest.ExtractCookies(loginCallbackResponse, []string{"access_token"})
	if err != nil {
		return token, err
	}

	token = cookies["access_token"]
	return token, err
}

func (k *keycloak) createLoginRequest(loginForm *http.Response) (*http.Request, url.Values, error) {
	var request *http.Request
	var loginParams url.Values
	var err error

	// extract cookies
	for _, c := range loginForm.Cookies() {
		if c.Name == "AUTH_SESSION_ID" {
			k.cookieAuthSessionID = c.Value
		}

		if c.Name == "KC_RESTART" {
			k.cookieKeycloakRestart = c.Value
		}
	}

	// figure out the login form parameter values
	htmlDoc, err := goquery.NewDocumentFromReader(loginForm.Body)
	if err != nil {
		return request, loginParams, err
	}

	form := htmlDoc.Find("form[id=kc-form-login]")
	loginActionURL, err := url.Parse(form.AttrOr("action", ""))
	if err != nil {
		return request, loginParams, err
	}

	loginParams = url.Values{}
	loginParams.Set("username", k.UsernameOrEmail)
	loginParams.Set("password", k.Password)
	loginParams.Set("login", "Log In")

	request, err = http.NewRequest("POST", loginActionURL.String(), strings.NewReader(loginParams.Encode()))
	if err != nil {
		return request, loginParams, err
	}

	oidctest.SetHeaders(request, map[string]string{
		"user-agent":   "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/67.0.3396.62 Safari/537.36",
		"accept":       "*/*",
		"content-type": "application/x-www-form-urlencoded",
		"cookie":       fmt.Sprintf("KC_RESTART=%s; AUTH_SESSION_ID=%s", k.cookieKeycloakRestart, k.cookieAuthSessionID),
	})

	return request, loginParams, err
}
