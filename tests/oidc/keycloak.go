package oidc

import (
	"fmt"
	"github.com/PuerkitoBio/goquery"
	"github.com/datawire/apro/lib/testutil"
	"net/http"
	"net/url"
	"strings"
)

type keycloak struct {
	*AuthenticationContext
	cookieAuthSessionID   string
	cookieKeycloakRestart string
}

func (k *keycloak) Authenticate(ctx *AuthenticationContext) (token string, err error) {
	assert := testutil.Assert{T: ctx.T}
	k.AuthenticationContext = ctx

	assert.HTTPResponseStatusEQ(k.initialAuthResponse, http.StatusOK)

	loginRequest, loginParams, err := k.createLoginRequest(k.initialAuthResponse)
	if err != nil {
		return
	}

	loginResponse, err := k.HTTP.Do(loginRequest)
	if err != nil {
		return
	}

	assert.HTTPResponseStatusEQ(loginResponse, http.StatusFound)
	cookies, err := ExtractCookies(loginResponse,
		[]string{"KC_RESTART", "KEYCLOAK_IDENTITY", "KEYCLOAK_SESSION", "KEYCLOAK_REMEMBER_ME"})

	if err != nil {
		return
	}

	loginCallbackRequest, err := http.NewRequest("POST", loginResponse.Header.Get("Location"), strings.NewReader(loginParams.Encode()))
	if err != nil {
		return
	}

	SetHeaders(loginCallbackRequest, map[string]string{
		"accept":       "*/*",
		"content-type": "application/x-www-form-urlencoded",
		"cookie":       FormatCookieHeaderFromCookieMap(cookies),
	})

	loginCallbackResponse, err := k.HTTP.Do(loginCallbackRequest)
	if err != nil {
		return
	}

	assert.HTTPResponseStatusEQ(loginCallbackResponse, http.StatusTemporaryRedirect)
	cookies, err = ExtractCookies(loginCallbackResponse, []string{"access_token"})
	if err != nil {
		return
	}

	token = cookies["access_token"]
	return
}

func (k *keycloak) createLoginRequest(loginForm *http.Response) (request *http.Request, loginParams url.Values, err error) {
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
		return
	}

	form := htmlDoc.Find("form[id=kc-form-login]")
	loginActionURL, err := url.Parse(form.AttrOr("action", ""))
	if err != nil {
		return
	}

	loginParams = url.Values{}
	loginParams.Set("username", k.UsernameOrEmail)
	loginParams.Set("password", k.Password)
	loginParams.Set("login", "Log In")

	request, err = http.NewRequest("POST", loginActionURL.String(), strings.NewReader(loginParams.Encode()))
	if err != nil {
		return
	}

	SetHeaders(request, map[string]string{
		"user-agent":   "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/67.0.3396.62 Safari/537.36",
		"accept":       "*/*",
		"content-type": "application/x-www-form-urlencoded",
		"cookie":       fmt.Sprintf("KC_RESTART=%s; AUTH_SESSION_ID=%s", k.cookieKeycloakRestart, k.cookieAuthSessionID),
	})

	return
}
