package main

import (
	"fmt"
	"github.com/PuerkitoBio/goquery"
	"net/http"
	"net/url"
	"strings"
)

type keycloak struct {
	*idp
	Username string
	Password string

	// Cookie values sent back by Keycloak in the login form. Purpose is opaque from our perspective but they seem
	// important.
	cookieAuthSessionID   string
	cookieKeycloakRestart string
}

func (keycloak *keycloak) AuthenticateV2(authRequest *http.Request, authResponse *http.Response) (token string, err error) {
	CheckIfStatus(authResponse, http.StatusOK)

	loginRequest, loginParams, err := keycloak.createLoginRequest(authResponse)
	if err != nil {
		return
	}

	loginResponse, err := keycloak.idp.httpClient.Do(loginRequest)
	if err != nil {
		return
	}
	CheckIfStatus(loginResponse, http.StatusFound)
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

	loginCallbackResponse, err := keycloak.idp.httpClient.Do(loginCallbackRequest)
	if err != nil {
		return
	}
	CheckIfStatus(loginCallbackResponse, http.StatusTemporaryRedirect)
	cookies, err = ExtractCookies(loginCallbackResponse, []string{"access_token"})
	if err != nil {
		return
	}

	token = cookies["access_token"]
	return
}

func (keycloak *keycloak) createLoginRequest(loginForm *http.Response) (request *http.Request, loginParams url.Values, err error) {
	// extract cookies
	for _, c := range loginForm.Cookies() {
		if c.Name == "AUTH_SESSION_ID" {
			keycloak.cookieAuthSessionID = c.Value
		}

		if c.Name == "KC_RESTART" {
			keycloak.cookieKeycloakRestart = c.Value
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
	loginParams.Set("username", keycloak.Username)
	loginParams.Set("password", keycloak.Password)
	loginParams.Set("login", "Log In")

	request, err = http.NewRequest("POST", loginActionURL.String(), strings.NewReader(loginParams.Encode()))
	if err != nil {
		return
	}

	request.Header.Add(
		"User-Agent",
		"Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/67.0.3396.62 Safari/537.36")

	request.Header.Set("accept", "*/*")
	request.Header.Set("accept-language", "en-US,en;q=0.9")
	request.Header.Set("content-type", "application/x-www-form-urlencoded")
	request.Header.Set("origin", "http://keycloak.localdev.svc.cluster.local")
	request.Header.Set("cookie", fmt.Sprintf("KC_RESTART=%s; AUTH_SESSION_ID=%s", keycloak.cookieKeycloakRestart, keycloak.cookieAuthSessionID))

	return
}
