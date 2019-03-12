package runner

// TODO(gagula): Missing the following tests.
// 1. Authorization token is expired.
// 2. Authorization token has invalid alg.
// 3. Authorization token has invalid aud.
// 4. Authorization token has invalid iss.
// 5. Request contains Client-ID and Secret-ID (assert that Secret header is deleted).
// 6. Cookie has valid token (requires signing token).
// 7. Callback endpoint when IDP response is negative or an error.
import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/datawire/apro/lib/testutil"
)

var appUT http.Handler
var idpSRV *httptest.Server
var appSRV *httptest.Server
var appClient *http.Client
var assert testutil.Assert

func setup(tb testing.TB) {
	assert = testutil.Assert{T: tb}

	idpSRV = NewIDP()
	appSRV, appUT = NewAPP(idpSRV.URL, tb)
	appClient = appSRV.Client()
	appClient.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		return http.ErrUseLastResponse
	}
}

func teardown(tb testing.TB) {
	appSRV.Close()
	idpSRV.Close()
}

// TestAppNoToken verifies the authorization server redirects the call to the IDP
// when the authorization header is empty.
func TestAppNoToken(t *testing.T) {
	setup(t)
	defer teardown(t)

	req, err := http.NewRequest("GET", appSRV.URL, nil)
	assert.NotError(err)

	res, err := appClient.Do(req)
	assert.NotError(err)
	assert.Bool(res != nil)

	assert.IntEQ(303, res.StatusCode)
	u, err := url.Parse(res.Header.Get("location"))
	assert.NotError(err)
	assert.StrEQ("friends", u.Query().Get("audience"))
	assert.StrEQ("code", u.Query().Get("response_type"))
	assert.StrEQ(fmt.Sprintf("%s/callback", appSRV.URL), u.Query().Get("redirect_uri"))
	assert.IntEQ(552, len(u.Query().Get("state")))
}

// TestAppBadToken verifies the authorization server returns 401 when the authorization
// header is invalid.
func TestAppBadToken(t *testing.T) {
	setup(t)
	defer teardown(t)

	req, err := http.NewRequest("GET", appSRV.URL, nil)
	assert.NotError(err)
	req.Header.Add("Authorization", "Bearer 1234")

	res, err := appClient.Do(req)
	assert.NotError(err)
	assert.Bool(res != nil)

	assert.IntEQ(303, res.StatusCode)
}

// TestAppBadCookie verifies the authorization server returns 401 when the authorization
// header is invalid.
func TestAppBadCookie(t *testing.T) {
	setup(t)
	defer teardown(t)

	req, err := http.NewRequest("GET", appSRV.URL, nil)
	assert.NotError(err)
	req.AddCookie(&http.Cookie{Name: "access_token", Value: "foo"})

	res, err := appClient.Do(req)
	assert.NotError(err)
	assert.Bool(res != nil)

	assert.IntEQ(303, res.StatusCode)
}

// TestAppCallback verifies the authorization server properly redirects when callback path
// is called with correct code and signed state. Verifies that the response from the
// IDP is parsed correctly.
func TestAppCallback(t *testing.T) {
	setup(t)
	defer teardown(t)

	// 1. We call the the authorization server (appSRV) just to get a signed state token
	// via /authorize redirect.
	req, err := http.NewRequest("GET", fmt.Sprintf("%s/foo", appSRV.URL), nil)
	assert.NotError(err)

	res, err := appClient.Do(req)
	assert.NotError(err)
	assert.Bool(res != nil)

	loc := res.Header.Get("location")
	assert.StrNotEmpty(loc)
	u, err := url.Parse(loc)
	assert.NotError(err)
	assert.IntEQ(556, len(u.Query().Get("state")))

	// 2. Now we call the authorization server (appSRV) again with a signed state and
	// code query params. Note that by calling it with code=authorize, our
	// fake IDP server (idpSRV) will respond with a mocked access_token.
	//
	// We check if the request contains the redirect with the original
	// request path `/foo`, client-id and cookie.
	req, err = http.NewRequest("GET", fmt.Sprintf("%s/callback?state=%s&code=authorize", appSRV.URL, u.Query().Get("state")), nil)
	assert.NotError(err)

	res, err = appClient.Do(req)
	assert.NotError(err)
	assert.Bool(res != nil)

	assert.IntEQ(307, res.StatusCode)
	assert.StrEQ(fmt.Sprintf("%s/foo", appSRV.URL), res.Header.Get("location"))
	cookie := res.Cookies()[0]
	assert.StrEQ("access_token", cookie.Name)
	assert.StrEQ("mocked_token_123", cookie.Value)
}

// TestAppCallback verifies that the authorization server return 401 when code is
// is not present in the request to the callback endpoint.
func TestAppCallbackNoCode(t *testing.T) {
	setup(t)
	defer teardown(t)

	// 1
	req, err := http.NewRequest("GET", fmt.Sprintf("%s/foo", appSRV.URL), nil)
	assert.NotError(err)

	res, err := appClient.Do(req)
	assert.NotError(err)
	assert.Bool(res != nil)

	u, err := url.Parse(res.Header.Get("location"))
	assert.NotError(err)
	assert.IntEQ(556, len(u.Query().Get("state")))

	// 2
	req, err = http.NewRequest("GET", fmt.Sprintf("%s/callback?state=%s", appSRV.URL, u.Query().Get("state")), nil)
	assert.NotError(err)

	res, err = appClient.Do(req)
	assert.NotError(err)
	assert.Bool(res != nil)

	assert.IntEQ(401, res.StatusCode)
}
