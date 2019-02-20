package main

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"net/http/cookiejar"
	"net/http/httputil"
	"net/url"
	"os"
	"time"
)

type idp struct {
	audience   string
	clientID   string
	scopes     []string
	httpClient *http.Client
}

type Authenticator interface {
	AuthenticateV2(request *http.Request, response *http.Response) (token string, err error)
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

func createHTTPRequest(method string, url url.URL) (*http.Request, error) {
	request, err := http.NewRequest(method, url.String(), nil)
	if err != nil {
		return nil, err
	}

	return request, nil
}

func main() {
	httpClient := newHTTPClient(5 * time.Second)

	providers := []interface{}{
		//&auth0{
		//	username: "testuser@datawire.com",
		//	password: "TestUser321",
		//	tenant:   "ambassador-oauth-e2e",
		//	idp: &idp{
		//		audience: "https://ambassador-oauth-e2e.auth0.com/api/v2/",
		//		clientID: "DOzF9q7U2OrvB7QniW9ikczS1onJgyiC",
		//		scopes:   []string{"openid", "profile", "email"},
		//		httpClient: &httpClient,
		//	},
		//},
		&keycloak{
			Username: "developer",
			Password: "developer",
			idp: &idp{
				audience:   "app",
				clientID:   "app",
				scopes:     []string{"openid"},
				httpClient: &httpClient,
			},
		},
	}

	for _, provider := range providers {
		handleIDP(&httpClient, provider)
	}
}

func handleIDP(httpClient *http.Client, v interface{}) {
	authenticator, ok := v.(Authenticator)
	if !ok {
		panic(v)
	}

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

	// We have been given a response from the authorization endpoint. This is IDP specific and could be the actual
	// login form OR just a redirect. At this point hand the code over to the IDP test implementation and let it
	// drive the authentication process.
	accessToken, err := authenticator.AuthenticateV2(authRequest, authResponse)
	CheckIfError(err)

	if accessToken == "" {
		fmt.Println("Error: Access Token was not returned by Identity Provider!")
		os.Exit(1)
	}

	// Almost home baby!
	unauthorizedRequest.Header.Add("Authorization", fmt.Sprintf("Bearer %s", accessToken))
	finalDestination, err := httpClient.Do(unauthorizedRequest)

	data, err := httputil.DumpResponse(finalDestination, true)
	CheckIfError(err)
	CheckIfStatus(finalDestination, http.StatusOK)

	fmt.Println(string(data))
}

//func main() {
//	auth0IDP := auth0{
//		username: "testuser@datawire.com",
//		password: "TestUser321",
//		tenant:   "ambassador-oauth-e2e",
//		idp: &idp{
//			audience: "https://ambassador-oauth-e2e.auth0.com/api/v2/",
//			clientID: "DOzF9q7U2OrvB7QniW9ikczS1onJgyiC",
//			scopes:   []string{"openid", "profile", "email"},
//		},
//	}
//
//	httpClient := newHTTPClient(5 * time.Second)
//	auth0IDP.httpClient = &httpClient
//
//	// 1. Initiate an HTTP GET request to our protected service
//	unauthorizedRequestURL := url.URL{
//		Scheme: "https",
//		Host:   "ambassador.localdev.svc.cluster.local",
//		Path:   "/httpbin/headers",
//	}
//
//	fmt.Println(1)
//	unauthorizedRequest, err := createHTTPRequest("GET", unauthorizedRequestURL)
//	CheckIfError(err)
//
//	fmt.Println(2)
//	unauthorizedResponse, err := httpClient.Do(unauthorizedRequest)
//	CheckIfError(err)
//	CheckIfStatus(unauthorizedResponse, http.StatusSeeOther)
//
//	// 2. Construct a redirect to the Identity Providers Authorization endpoint. Since we do not have an access token
//	// at this point we will end up being redirected.
//	redirectURL, err := url.Parse(unauthorizedResponse.Header.Get("Location"))
//	CheckIfError(err)
//
//	fmt.Println(3)
//	authRequest, err := createHTTPRequest("GET", *redirectURL)
//	CheckIfError(err)
//
//	fmt.Println(4)
//	authResponse, err := httpClient.Do(authRequest)
//	CheckIfError(err)
//
//	// We have been given a response from the authorization endpoint. This is IDP specific and could be the actual
//	// login form OR just a redirect. At this point hand the code over to the IDP test implementation and let it
//	// drive the authentication process.
//	accessToken, err := auth0IDP.AuthenticateV2(authRequest, authResponse)
//
//	if accessToken == "" {
//		fmt.Println("Error: Access Token was not returned by Identity Provider!")
//		os.Exit(1)
//	}
//
//	// Almost home baby!
//	unauthorizedRequest.Header.Add("Authorization", fmt.Sprintf("Bearer %s", accessToken))
//	finalDestination, err := httpClient.Do(unauthorizedRequest)
//
//	data, err := httputil.DumpResponse(finalDestination, true)
//	CheckIfError(err)
//	CheckIfStatus(finalDestination, http.StatusOK)
//
//	fmt.Println(string(data))
//}
//
//func main() {
//
//}
