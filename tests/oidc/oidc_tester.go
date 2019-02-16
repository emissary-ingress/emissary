package main

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"github.com/PuerkitoBio/goquery"
	"net/http"
	"net/http/cookiejar"
	"net/http/httputil"
	"net/url"
	"os"
	"strings"
	"time"
)

type oidcIntegration interface {
	createLoginRequest() (*http.Request, error)
	doLogin() (string, error)
}

type RequestResponsePair struct {
	Request  *http.Request
	Response *http.Response
}

type History struct {
	entries map[string]RequestResponsePair
}

func (h *History) registerRequest(name string, pair RequestResponsePair) {

}

func (h *History) getUnauthorizedPair() *RequestResponsePair {
	return nil
}

// =====================================
// Auth0 IdP
// =====================================

type auth0 struct {
	username   string
	password   string
	tenant     string
	audience   string
	clientID   string
	scopes     []string
	csrf       string
	state      string
	httpClient *http.Client
}

// auth0Login is the payload sent to the Auth0 Username and Password login endpoint.
type auth0UsernameAndPasswordLogin struct {
	Audience     string                 `json:"audience"`
	ClientID     string                 `json:"client_id"`
	Connection   string                 `json:"connection"`
	CSRF         string                 `json:"_csrf"`
	Intstate     string                 `json:"_intstate"`
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

func (auth0 *auth0) Authenticate(loginForm *http.Response, state string) (string, error) {
	loginRequest, err := auth0.createLoginRequest(loginForm, state)
	CheckIfError(err)

	loginRequest.Header.Add(
		"User-Agent",
		"Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/67.0.3396.62 Safari/537.36")

	loginResponse, err := auth0.httpClient.Do(loginRequest)
	CheckIfError(err)
	CheckIfStatus(loginResponse, 200)

	htmlDoc, err := goquery.NewDocumentFromReader(loginResponse.Body)
	CheckIfError(err)

	form := htmlDoc.Find("form[name=hiddenform]")
	loginCallbackURL, err := url.Parse(form.AttrOr("action", ""))
	CheckIfError(err)

	loginCallbackToken := htmlDoc.Find("input[name=wresult]").AttrOr("value", "")
	loginCallbackCtx := htmlDoc.Find("input[name=wctx]").AttrOr("value", "")

	formData := url.Values{}
	formData.Add("wresult", loginCallbackToken)
	formData.Add("wctx", loginCallbackCtx)

	loginCallbackRequest, err := http.NewRequest("POST", loginCallbackURL.String(), strings.NewReader(formData.Encode()))
	// this is all stuff the browser sends so add it as well in case its needed
	loginCallbackRequest.Header.Set("accept", "*/*")
	loginCallbackRequest.Header.Set("accept-language", "en-US,en;q=0.9")
	loginCallbackRequest.Header.Set("content-type", "application/x-www-form-urlencoded")
	loginCallbackRequest.Header.Set("dnt", "1")
	loginCallbackRequest.Header.Set("origin", fmt.Sprintf("https://%s.auth0.com/", auth0.tenant))
	loginCallbackRequest.Header.Set("referrer", "https://ambassador-oauth-e2e.auth0.com/login")
	loginCallbackRequest.Header.Set("cookie", fmt.Sprintf("_csrf=%s", auth0.csrf)) // VERY IMPORTANT. 403 if excluded

	loginCallbackResponse, err := auth0.httpClient.Do(loginCallbackRequest)
	CheckIfError(err)

	//data, err := httputil.DumpResponse(loginCallbackResponse, true)
	//fmt.Println(string(data))

	// back to our callback...
	redirectURL, err := url.Parse(loginCallbackResponse.Header.Get("Location"))
	callbackRequest, err := http.NewRequest("POST", redirectURL.String(), strings.NewReader(formData.Encode()))
	callbackRequest.Header.Set("accept", "*/*")
	callbackRequest.Header.Set("accept-language", "en-US,en;q=0.9")
	callbackRequest.Header.Set("content-type", "application/x-www-form-urlencoded")
	callbackRequest.Header.Set("dnt", "1")
	callbackRequest.Header.Set("origin", fmt.Sprintf("https://%s.auth0.com/", auth0.tenant))
	callbackRequest.Header.Set("referrer", "https://ambassador-oauth-e2e.auth0.com/login")
	callbackRequest.Header.Set("cookie", fmt.Sprintf("_csrf=%s", auth0.csrf)) // VERY IMPORTANT. 403 if excluded

	callbackResponse, err := auth0.httpClient.Do(callbackRequest)
	if err != nil {
		panic(err)
	}

	accessToken := ""
	for _, c := range callbackResponse.Cookies() {
		if c.Name == "access_token" {
			accessToken = c.Value
			break
		}
	}

	return accessToken, nil
}

func (idp *auth0) createLoginRequest(r *http.Response, state string) (*http.Request, error) {

	// get the csrf token
	for _, c := range r.Cookies() {
		if c.Name == "_csrf" {
			idp.csrf = c.Value
		}
	}

	loginEndpoint, err := url.Parse(fmt.Sprintf("https://%s.auth0.com/usernamepassword/login", idp.tenant))
	if err != nil {
		return nil, err
	}

	loginParams := auth0UsernameAndPasswordLogin{
		Audience:     idp.audience,
		ClientID:     idp.clientID,
		CSRF:         idp.csrf,
		Connection:   "Username-Password-Authentication",
		Intstate:     "deprecated",
		Password:     idp.password,
		PopupOptions: make(map[string]interface{}),
		Protocol:     "oauth2",
		RedirectURI:  "https://ambassador.localdev.svc.cluster.local/callback",
		ResponseType: "code",
		Scope:        strings.Join(idp.scopes, " "),
		State:        state,
		SSO:          true,
		Tenant:       idp.tenant,
		Username:     idp.username,
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
	request.Header.Set("accept", "*/*")
	request.Header.Set("accept-language", "en-US,en;q=0.9")
	request.Header.Set("content-type", "application/json")
	request.Header.Set("dnt", "1")
	request.Header.Set("origin", fmt.Sprintf("https://%s.auth0.com/", idp.tenant))
	request.Header.Set("referrer", "https://ambassador-oauth-e2e.auth0.com/login")
	request.Header.Set("cookie", fmt.Sprintf("_csrf=%s", idp.csrf)) // VERY IMPORTANT. 403 if excluded

	// this is a JSON object which carries some version info about the login UI implementation...
	request.Header.Set("auth0-client", "eyJuYW1lIjoibG9jay5qcyIsInZlcnNpb24iOiIxMS4xMS4wIiwibGliX3ZlcnNpb24iOnsicmF3IjoiOS44LjEifX0=")

	return request, nil
}

// =====================================
// Keycloak IdP
// =====================================

type keycloak struct{}

func (idp *keycloak) fmtLoginURL() url.URL {
	return url.URL{}
}

// returns an http client that is configured for use in OpenID Connect authentication tests. The client is configured
// to ignore self-signed TLS certificates and to not follow redirects automatically.
func newHTTPClient(timeout time.Duration) http.Client {
	cookieJar, _ := cookiejar.New(nil)
	return http.Client{
		// DO NOT FOLLOW REDIRECTS: https://stackoverflow.com/a/38150816
		//
		// This is test code. We do not want to follow any redirects automatically because we may want to write
		// assertions against those responses.
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},

		// Disable HTTPS certificate validation for this client because we are likely using self-signed certificates
		// during tests.
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},

		Jar:     cookieJar,
		Timeout: timeout,
	}
}

func CheckIfError(err error) {
	if err == nil {
		return
	}

	fmt.Printf("Error %s", err)
	os.Exit(1)
}

func CheckIfStatus(r *http.Response, expectedStatus int) {
	if r.StatusCode != expectedStatus {
		fmt.Printf("Error expected HTTP status %d but was %d\n", expectedStatus, r.StatusCode)

		data, err := httputil.DumpResponse(r, true)
		CheckIfError(err)
		fmt.Println(string(data))

		os.Exit(1)
	}
}

func main() {
	auth0 := auth0{
		username: "testuser@datawire.com",
		password: "TestUser321",
		tenant:   "ambassador-oauth-e2e",
		scopes:   []string{"openid", "profile", "email"},
		audience: "https://ambassador-oauth-e2e.auth0.com/api/v2/",
		clientID: "DOzF9q7U2OrvB7QniW9ikczS1onJgyiC",
	}

	httpClient := newHTTPClient(5 * time.Second)
	auth0.httpClient = &httpClient

	// 1. Initiate an HTTP GET request to our protected service
	unauthorizedRequestURL := url.URL{
		Scheme: "https",
		Host:   "ambassador.localdev.svc.cluster.local",
		Path:   "/httpbin/headers",
	}

	unauthorizedRequest, err := createHTTPRequest("GET", unauthorizedRequestURL)
	CheckIfError(err)

	unauthorizedResponse, err := httpClient.Do(unauthorizedRequest)
	CheckIfError(err)
	CheckIfStatus(unauthorizedResponse, http.StatusSeeOther)

	// 2. Construct a redirect to the Identity Providers Authorization endpoint. Since we do not have an access token
	// at this point we will end up being redirected.
	redirectURL, err := url.Parse(unauthorizedResponse.Header.Get("Location"))
	CheckIfError(err)

	authRequest, err := createHTTPRequest("GET", *redirectURL)
	CheckIfError(err)

	authResponse, err := httpClient.Do(authRequest)
	CheckIfError(err)
	CheckIfStatus(authResponse, http.StatusFound)

	// 3. Handle the Redirect and goto the IdP Login URL
	loginUIRedirectURL, err := url.Parse(authResponse.Header.Get("Location"))

	// Auth0 hands back relative (path-only) redirects which are useless for subsequent requests. We are talking to the
	// same endpoint as "authRequest" so just use the scheme, host and port info from that.
	loginUIRedirectURL.Scheme = authRequest.URL.Scheme
	loginUIRedirectURL.Host = authRequest.URL.Host
	loginUIRequest, err := createHTTPRequest("GET", *loginUIRedirectURL)
	CheckIfError(err)

	loginUIResponse, err := httpClient.Do(loginUIRequest)
	CheckIfError(err)
	CheckIfStatus(loginUIResponse, http.StatusOK)

	// We should be at a login form by this point. Unfortunately the login form rendered is IDP specific, for
	// example, Auth0 renders it with some JavaScript magic. This form has important attributes on it like the form
	// field names to send during the login request.
	accessToken, err := auth0.Authenticate(loginUIResponse, loginUIRedirectURL.Query().Get("state"))
	CheckIfError(err)

	if accessToken == "" {
		fmt.Println("Access Token was not returned")
		os.Exit(1)
	}

	unauthorizedRequest.Header.Add("Authorization", fmt.Sprintf("Bearer %s", accessToken))
	finalDestination, err := httpClient.Do(unauthorizedRequest)

	data, err := httputil.DumpResponse(finalDestination, true)
	CheckIfError(err)
	CheckIfStatus(finalDestination, 200)

	fmt.Println(string(data))
}

func createHTTPRequest(method string, url url.URL) (*http.Request, error) {
	request, err := http.NewRequest(method, url.String(), nil)
	if err != nil {
		return nil, err
	}

	return request, nil
}
