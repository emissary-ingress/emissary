package main

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"github.com/PuerkitoBio/goquery"
	"github.com/davecgh/go-spew/spew"
	"net/http"
	"net/http/cookiejar"
	"net/http/httputil"
	"net/url"
	"strings"
	"time"
)

type oidcIntegration interface {
	createLoginRequest() (*http.Request, error)
}

// =====================================
// Auth0 IdP
// =====================================

type auth0 struct {
	username string
	password string
	tenant   string
	audience string
	clientID string
	scopes   []string
	csrf     string
	state    string
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
	fmt.Println(loginParamsString)

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

//func (c *Rest) request(method string, url fmt.Stringer, params url.Values, body interface{}) (*http.Request, error) {
//	rq, err := http.NewRequest(method, url.String(), strings.NewReader(params.Encode()))
//	if err != nil {
//		return nil, err
//	}
//
//	rq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
//	if c.token != "" {
//		rq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.token))
//	}
//
//	return rq, nil
//}

// =====================================
// Keycloak IdP
// =====================================

type keycloak struct{}

func (idp *keycloak) fmtLoginURL() url.URL {
	return url.URL{}
}

func main() {
	var err error

	cookieJar, _ := cookiejar.New(nil)
	httpClient := http.Client{
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
		Timeout: 5 * time.Second,
	}

	// 1. Initiate an HTTP GET request to our protected service
	unauthorizedRequestURL := url.URL{
		Scheme: "https",
		Host:   "ambassador.localdev.svc.cluster.local",
		Path:   "/httpbin/headers",
	}

	unauthorizedRequest, err := createHTTPRequest("GET", unauthorizedRequestURL)
	if err != nil {
		panic(err)
	}

	unauthorizedResponse, err := httpClient.Do(unauthorizedRequest)
	if err != nil {
		panic(err)
	}

	// 2. Check that we have been given a Redirect
	if unauthorizedResponse.StatusCode != 303 {
		panic(fmt.Errorf("unexpected HTTP status code for initial redirect: %d", unauthorizedResponse.StatusCode))
	}

	// 3. Construct an HTTP request to the IDP Authorization URL.
	redirectURL, err := url.Parse(unauthorizedResponse.Header.Get("Location"))

	//// this will be needed because future requests will give us relative URLs
	//redirectURLBase := url.URL{
	//	Scheme: redirectURL.Scheme,
	//	Host: redirectURL.Host,
	//}

	loginUIRequest, err := createHTTPRequest("GET", *redirectURL)
	if err != nil {
		panic(err)
	}

	// 4. Since we're not Authn/z with the IdP yet we *SHOULD* be redirected to a Login UI
	loginUIResponse, err := httpClient.Do(loginUIRequest)
	if err != nil {
		panic(err)
	}

	if loginUIResponse.StatusCode != 302 {
		spew.Dump(loginUIResponse)
		panic(fmt.Errorf("unexpected HTTP status code for login UI redirect %d", unauthorizedResponse.StatusCode))
	}

	// 5. Construct a request to the IdP login UI based
	redirectURL, err = url.Parse(loginUIResponse.Header.Get("Location"))
	redirectURL.Scheme = "https"
	redirectURL.Host = loginUIRequest.Host

	loginUIRequest, err = createHTTPRequest("GET", *redirectURL)
	if err != nil {
		panic(err)
	}

	loginUIResponse, err = httpClient.Do(loginUIRequest)
	if err != nil {
		panic(err)
	}

	if loginUIResponse.StatusCode != 200 {
		spew.Dump(loginUIResponse)
		panic(fmt.Errorf("unexpected HTTP status code after login UI redirect %d", unauthorizedResponse.StatusCode))
	}

	// We should be at a login form by this point. Unfortunately the login form is rendered is IDP specific, for
	// example, Auth0 renders it with some JavaScript magic. This form has important attributes on it like the form
	// field names to send during the login request.

	auth0 := auth0{
		username: "testuser@datawire.com",
		password: "TestUser321",
		tenant:   "ambassador-oauth-e2e",
		scopes:   []string{"openid", "profile", "email"},
		audience: "https://ambassador-oauth-e2e.auth0.com/api/v2/",
		clientID: "DOzF9q7U2OrvB7QniW9ikczS1onJgyiC",
	}

	loginRequest, err := auth0.createLoginRequest(loginUIResponse, redirectURL.Query().Get("state"))

	// Set the User-Agent to be Chrome just in case the IDPs do something like inspect the User Agent
	loginRequest.Header.Add(
		"User-Agent",
		"Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/67.0.3396.62 Safari/537.36")

	if err != nil {
		panic(err)
	}

	loginResponse, err := httpClient.Do(loginRequest)
	if loginResponse.StatusCode != 200 {
		data, _ := httputil.DumpResponse(loginResponse, true)
		fmt.Println(string(data))
		panic(fmt.Errorf("unexpected HTTP status code after login %d", loginResponse.StatusCode))
	}

	htmlDoc, err := goquery.NewDocumentFromReader(loginResponse.Body)
	if err != nil {
		panic(err)
	}

	form := htmlDoc.Find("form[name=hiddenform]")
	loginCallbackURL, err := url.Parse(form.AttrOr("action", ""))
	if err != nil {
		panic(err)
	}

	loginCallbackToken := htmlDoc.Find("input[name=wresult]").AttrOr("value", "")
	loginCallbackCtx := htmlDoc.Find("input[name=wctx]").AttrOr("value", "")

	fmt.Printf("Callback URL: %s\n", loginCallbackURL)
	fmt.Printf("Callback Token: %s\n", loginCallbackToken)
	fmt.Printf("Callback Context: %s\n", loginCallbackCtx)

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

	loginCallbackResponse, err := httpClient.Do(loginCallbackRequest)
	if err != nil {
		panic(err)
	}

	data, err := httputil.DumpResponse(loginCallbackResponse, true)
	fmt.Println(string(data))

	// back to our callback...
	redirectURL, err = url.Parse(loginCallbackResponse.Header.Get("Location"))
	callbackRequest, err := http.NewRequest("POST", redirectURL.String(), strings.NewReader(formData.Encode()))
	callbackRequest.Header.Set("accept", "*/*")
	callbackRequest.Header.Set("accept-language", "en-US,en;q=0.9")
	callbackRequest.Header.Set("content-type", "application/x-www-form-urlencoded")
	callbackRequest.Header.Set("dnt", "1")
	callbackRequest.Header.Set("origin", fmt.Sprintf("https://%s.auth0.com/", auth0.tenant))
	callbackRequest.Header.Set("referrer", "https://ambassador-oauth-e2e.auth0.com/login")
	callbackRequest.Header.Set("cookie", fmt.Sprintf("_csrf=%s", auth0.csrf)) // VERY IMPORTANT. 403 if excluded

	callbackResponse, err := httpClient.Do(callbackRequest)
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

	data, err = httputil.DumpResponse(callbackResponse, true)
	fmt.Println(string(data))

	unauthorizedRequest.Header.Add("Authorization", fmt.Sprintf("Bearer %s", accessToken))
	finalDestination, err := httpClient.Do(unauthorizedRequest)

	data, err = httputil.DumpResponse(finalDestination, true)
	fmt.Println(string(data))
}

func createHTTPRequest(method string, url url.URL) (*http.Request, error) {
	request, err := http.NewRequest(method, url.String(), nil)
	if err != nil {
		return nil, err
	}

	//rq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	//if c.token != "" {
	//	rq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.token))
	//}

	return request, nil
}
