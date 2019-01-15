package main

// TODO(gagula): Missing the following tests.
// 1. Authorization token is expired.
// 2. Authorization token has invalid alg.
// 3. Authorization token has invalid aud.
// 4. Authorization token has invalid iss.
// 5. Request contains Client-ID and Secret-ID (assert that Secret header is deleted).
// 6. Cookie has valid token (requires signing token).
// 7. Callback endpoint when IDP response is negative or an error.
import (
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"testing"

	"github.com/datawire/ambassador-oauth/cmd/ambassador-oauth/app"
	"github.com/datawire/ambassador-oauth/testutil"
)

var appUT *app.App
var idpSRV *httptest.Server
var appSRV *httptest.Server
var appClient *http.Client
var idpClient *http.Client

func TestMain(m *testing.M) {
	// Setup Test Servers & Clients
	idpSRV = testutil.NewIDP()
	appSRV, appUT = testutil.NewAPP(idpSRV.URL)
	appClient = appSRV.Client()

	// Run
	ok := m.Run()

	// Teardown
	appSRV.Close()
	idpSRV.Close()

	// Exit
	os.Exit(ok)
}

// TestAppNoToken verifies the authorization server redirects the call to the IDP
// when the authorization header is empty.
func TestAppNoToken(t *testing.T) {
	assert := testutil.Assert{T: t}

	appClient.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		return errors.New("")
	}

	req, _ := http.NewRequest("GET", appSRV.URL, nil)
	res, _ := appClient.Do(req)
	u, _ := url.Parse(res.Header.Get("location"))

	assert.StrEQ("friends", u.Query().Get("audience"))
	assert.StrEQ("code", u.Query().Get("response_type"))
	assert.StrEQ(fmt.Sprintf("%s/callback", idpSRV.URL), u.Query().Get("redirect_uri"))
	assert.IntEQ(303, res.StatusCode)
	assert.IntEQ(552, len(u.Query().Get("state")))
}

// TestAppBadToken verifies the authorization server returns 401 when the authorization
// header is invalid.
func TestAppBadToken(t *testing.T) {
	assert := testutil.Assert{T: t}

	req, _ := http.NewRequest("GET", appSRV.URL, nil)
	req.Header.Add("Authorization", "Bearer 1234")

	res, _ := appClient.Do(req)

	assert.NotNil(res)
	assert.IntEQ(303, res.StatusCode)
}

// TestAppBadCookie verifies the authorization server returns 401 when the authorization
// header is invalid.
func TestAppBadCookie(t *testing.T) {
	assert := testutil.Assert{T: t}

	req, _ := http.NewRequest("GET", appSRV.URL, nil)
	req.AddCookie(&http.Cookie{Name: "access_token", Value: "foo"})

	res, _ := appClient.Do(req)

	assert.NotNil(res)
	assert.IntEQ(303, res.StatusCode)
}

// TestAppCallback verifies the authorization server properly redirects when callback path
// is called with correct code and signed state. Verifies that the response from the
// IDP is parsed correctly.
func TestAppCallback(t *testing.T) {
	assert := testutil.Assert{T: t}

	// 1. We call the the authorization server (appSRV) just to get a signed state token
	// via /authorize redirect.
	appClient.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		return errors.New("")
	}

	// http://ip:port/foo
	reqURL := fmt.Sprintf("%s/foo", appSRV.URL)

	req, _ := http.NewRequest("GET", reqURL, nil)
	res, _ := appClient.Do(req)

	loc := res.Header.Get("location")
	assert.StrNotEmpty(loc)

	u, err := url.Parse(loc)
	assert.Nil(err)
	assert.IntEQ(556, len(u.Query().Get("state")))

	// 2. Now we call the authorization server (appSRV) again with a signed state and
	// code query params. Note that by calling it with code=authorize, our
	// fake IDP server (idpSRV) will respond with a mocked access_token.
	callbackURL := fmt.Sprintf("%s/callback?state=%s&code=authorize", appSRV.URL, u.Query().Get("state"))
	callbackREQ, _ := http.NewRequest("GET", callbackURL, nil)
	callbackRES, _ := appClient.Do(callbackREQ)

	// 3. Finally we check if the request contains the redirect with the original
	// request path `/foo`, client-id and cookie.
	assert.NotNil(callbackRES)
	assert.IntEQ(307, callbackRES.StatusCode)
	assert.StrEQ(reqURL, callbackRES.Header.Get("location"))
	cookie := callbackRES.Cookies()[0]
	assert.StrEQ("access_token", cookie.Name)
	assert.StrEQ("mocked_token_123", cookie.Value)
}

// TestAppCallback verifies that the authorization server return 401 when code is
// is not present in the request to the callback endpoint.
func TestAppCallbackNoCode(t *testing.T) {
	assert := testutil.Assert{T: t}

	appClient.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		return errors.New("")
	}

	reqURL := fmt.Sprintf("%s/foo", appSRV.URL)
	req, _ := http.NewRequest("GET", reqURL, nil)
	res, _ := appClient.Do(req)
	u, _ := url.Parse(res.Header.Get("location"))

	assert.IntEQ(556, len(u.Query().Get("state")))

	callbackURL := fmt.Sprintf("%s/callback?state=%s", appSRV.URL, u.Query().Get("state"))
	callbackREQ, _ := http.NewRequest("GET", callbackURL, nil)
	callbackRES, _ := appClient.Do(callbackREQ)

	assert.NotNil(callbackRES)
	assert.IntEQ(401, callbackRES.StatusCode)
}
