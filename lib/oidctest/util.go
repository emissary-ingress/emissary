package oidctest

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"
	"time"

	testutil "github.com/datawire/apro/lib/testutil"
)

func SetHeaders(r *http.Request, headers map[string]string) {
	for k, v := range headers {
		r.Header.Set(k, v)
	}
}

func FormatCookieHeaderFromCookieMap(cookies map[string]string) (result string) {
	for k, v := range cookies {
		result += fmt.Sprintf("%s=%s;", k, v)
	}

	result = strings.TrimSuffix(result, ";")

	return
}

func ExtractCookies(response *http.Response, names []string) (result map[string]string, err error) {
	result = make(map[string]string)

	for _, cookie := range response.Cookies() {
		if pos, contained := Contains(cookie.Name, names); contained {
			result[cookie.Name] = cookie.Value
			Remove(pos, names)
		}
	}

	//if len(names) != 0 {
	//	err = fmt.Errorf("not all cookies found: %v\n", names)
	//}

	return
}

func Contains(value string, items []string) (int, bool) {
	for idx, item := range items {
		if item == value {
			return idx, true
		}
	}

	return -1, false
}

func Remove(pos int, src []string) []string {
	src[len(src)-1], src[pos] = src[pos], src[len(src)-1]
	return src[:len(src)-1]
}

// returns an http client that is configured for use in OpenID Connect authentication tests. The client is configured
// to ignore self-signed TLS certificates and to not follow redirects automatically.
func NewHTTPClient(timeout time.Duration, enableCookies bool) http.Client {
	client := http.Client{
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
			/* #nosec */
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},

		Timeout: timeout,
	}

	if enableCookies {
		jar, _ := cookiejar.New(nil)
		client.Jar = jar
	}

	return client
}

func CreateHTTPRequest(method string, url url.URL) (*http.Request, error) {
	request, err := http.NewRequest(method, url.String(), nil)
	if err != nil {
		return nil, err
	}

	return request, nil
}

// testIDP performs the full standard authentication flow.
func TestIDP(ctx *AuthenticationContext) {
	assert := testutil.Assert{T: ctx.T}

	authenticator, ok := ctx.Authenticator.(AuthenticationDriver)
	if !ok {
		ctx.T.Fatal("Authenticator interface is not implemented")
	}

	// 1. Initiate an HTTP GET request to our protected service
	unauthorizedRequest, err := CreateHTTPRequest("GET", ctx.ProtectedResource)
	assert.NotError(err)

	unauthorizedResponse, err := ctx.HTTP.Do(unauthorizedRequest)
	assert.NotError(err)
	assert.HTTPResponseStatusEQ(unauthorizedResponse, http.StatusSeeOther)

	// 2. Construct a redirect to the Identity Providers Authorization endpoint. Since we do not have an access token
	// at this point we will end up being redirected.
	redirectURL, err := url.Parse(unauthorizedResponse.Header.Get("Location"))
	assert.NotError(err)

	authRequest, err := CreateHTTPRequest("GET", *redirectURL)
	assert.NotError(err)

	authResponse, err := ctx.HTTP.Do(authRequest)
	assert.NotError(err)

	// We have been given a response from the authorization endpoint. This is IDP specific and could be the actual
	// login form OR just a redirect. At this point hand the code over to the IDP test implementation and let it
	// drive the authentication process.
	ctx.InitialAuthRequest = authRequest
	ctx.InitialAuthResponse = authResponse

	accessToken, err := authenticator.Authenticate(ctx)
	assert.NotError(err)
	assert.StrNotEmpty(accessToken)

	// Almost home baby!
	ctx.HTTP.Jar = nil

	unauthorizedRequest.Header.Add("Authorization", fmt.Sprintf("Bearer %s", accessToken))

	// Create a new HTTP client that is entirely unused. This ensures the CookieJar is not re-used. A problem that was
	// discovered during development is that auth0 sets the "access_token" cookie which means the "Authorization"
	// header was being ignored.
	virginHTTPClient := NewHTTPClient(5*time.Second, false)

	finalDestination, err := virginHTTPClient.Do(unauthorizedRequest)
	assert.NotError(err)
	assert.HTTPResponseStatusEQ(finalDestination, http.StatusOK)
}
