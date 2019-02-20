package main

import (
	"encoding/json"
	"fmt"
	"github.com/PuerkitoBio/goquery"
	"net/http"
	"net/url"
	"strings"
)

type auth0 struct {
	*idp
	username string
	password string
	tenant   string
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

func (auth0 *auth0) AuthenticateV2(authRequest *http.Request, authResponse *http.Response) (token string, err error) {
	CheckIfStatus(authResponse, http.StatusFound)

	// 3. Handle the Redirect and goto the IdP Login URL
	loginUIRedirectURL, err := url.Parse(authResponse.Header.Get("Location"))

	// Auth0 hands back relative (path-only) redirects which are useless for subsequent requests. We are talking to the
	// same endpoint as "authRequest" so just use the scheme, host and port info from that.
	loginUIRedirectURL.Scheme = authRequest.URL.Scheme
	loginUIRedirectURL.Host = authRequest.URL.Host
	loginUIRequest, err := createHTTPRequest("GET", *loginUIRedirectURL)
	CheckIfError(err)

	loginUIResponse, err := auth0.httpClient.Do(loginUIRequest)
	CheckIfError(err)
	CheckIfStatus(loginUIResponse, http.StatusOK)

	loginForm := loginUIResponse

	loginRequest, err := auth0.createLoginRequest(loginForm, loginUIRedirectURL.Query().Get("state"))
	CheckIfError(err)

	loginRequest.Header.Add(
		"User-Agent",
		"Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/67.0.3396.62 Safari/537.36")

	loginResponse, err := auth0.idp.httpClient.Do(loginRequest)
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

	for _, c := range callbackResponse.Cookies() {
		if c.Name == "access_token" {
			token = c.Value
			break
		}
	}

	return
}

func (auth0 *auth0) Authenticate(loginForm *http.Response, state string) (token string, err error) {
	loginRequest, err := auth0.createLoginRequest(loginForm, state)
	CheckIfError(err)

	loginRequest.Header.Add(
		"User-Agent",
		"Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/67.0.3396.62 Safari/537.36")

	loginResponse, err := auth0.idp.httpClient.Do(loginRequest)
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

	for _, c := range callbackResponse.Cookies() {
		if c.Name == "access_token" {
			token = c.Value
			break
		}
	}

	return
}

func (auth0 *auth0) createLoginRequest(r *http.Response, state string) (*http.Request, error) {

	// get the csrf token
	for _, c := range r.Cookies() {
		if c.Name == "_csrf" {
			auth0.csrf = c.Value
		}
	}

	loginEndpoint, err := url.Parse(fmt.Sprintf("https://%s.auth0.com/usernamepassword/login", auth0.tenant))
	if err != nil {
		return nil, err
	}

	loginParams := auth0UsernameAndPasswordLogin{
		Audience:     auth0.audience,
		ClientID:     auth0.clientID,
		CSRF:         auth0.csrf,
		Connection:   "Username-Password-Authentication",
		Intstate:     "deprecated",
		Password:     auth0.password,
		PopupOptions: make(map[string]interface{}),
		Protocol:     "oauth2",
		RedirectURI:  "https://ambassador.localdev.svc.cluster.local/callback",
		ResponseType: "code",
		Scope:        strings.Join(auth0.scopes, " "),
		State:        state,
		SSO:          true,
		Tenant:       auth0.tenant,
		Username:     auth0.username,
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
	request.Header.Set("origin", fmt.Sprintf("https://%s.auth0.com/", auth0.tenant))
	request.Header.Set("referrer", "https://ambassador-oauth-e2e.auth0.com/login")
	request.Header.Set("cookie", fmt.Sprintf("_csrf=%s", auth0.csrf)) // VERY IMPORTANT. 403 if excluded

	// this is a JSON object which carries some version info about the login UI implementation...
	request.Header.Set("auth0-client", "eyJuYW1lIjoibG9jay5qcyIsInZlcnNpb24iOiIxMS4xMS4wIiwibGliX3ZlcnNpb24iOnsicmF3IjoiOS44LjEifX0=")

	return request, nil
}
