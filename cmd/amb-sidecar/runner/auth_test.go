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
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/datawire/apro/lib/filterapi"
	"github.com/datawire/apro/lib/testutil"
)

var idpServer *httptest.Server
var filterServer *httptest.Server
var filterClient filterapi.FilterClient
var assert testutil.Assert

func setup(tb testing.TB) {
	assert = testutil.Assert{T: tb}

	idpServer = NewIDP()
	filterServer, filterClient = NewAPP(idpServer.URL, tb)
}

func teardown(tb testing.TB) {
	filterServer.Close()
	idpServer.Close()
}

// TestFilterNoToken verifies the authorization server redirects the call to the IDP
// when the authorization header is empty.
func TestFilterNoToken(t *testing.T) {
	setup(t)
	defer teardown(t)

	res, err := filterClient.Filter(context.Background(), newFilterRequest("GET", "/", nil))
	assert.NotError(err)
	assert.Bool(res != nil)

	hres, hresOK := res.(*filterapi.HTTPResponse)
	if !hresOK {
		t.Fatalf("Expected %T, got %T", &filterapi.HTTPResponse{}, res)
	}
	assert.IntEQ(303, hres.StatusCode)
	u, err := url.Parse(hres.Header.Get("location"))
	assert.NotError(err)
	assert.StrEQ("friends", u.Query().Get("audience"))
	assert.StrEQ("code", u.Query().Get("response_type"))
	assert.StrEQ("http://lukeshu.com/callback", u.Query().Get("redirect_uri"))
	assert.IntEQ(547, len(u.Query().Get("state")))
}

// TestFilterBadToken verifies the authorization server returns 401 when the authorization
// header is invalid.
func TestFilterBadToken(t *testing.T) {
	setup(t)
	defer teardown(t)

	res, err := filterClient.Filter(context.Background(), newFilterRequest("GET", "/", http.Header{
		"Authorization": {"Bearer 1234"},
	}))
	assert.NotError(err)
	assert.Bool(res != nil)

	hres, hresOK := res.(*filterapi.HTTPResponse)
	if !hresOK {
		t.Fatalf("Expected %T, got %T", &filterapi.HTTPResponse{}, res)
	}
	assert.IntEQ(303, hres.StatusCode)
}

// TestFilterBadCookie verifies the authorization server returns 401 when the authorization
// header is invalid.
func TestFilterBadCookie(t *testing.T) {
	setup(t)
	defer teardown(t)

	res, err := filterClient.Filter(context.Background(), newFilterRequest("GET", "/", http.Header{
		"Cookie": {"access_token=foo"},
	}))
	assert.NotError(err)
	assert.Bool(res != nil)

	hres, hresOK := res.(*filterapi.HTTPResponse)
	if !hresOK {
		t.Fatalf("Expected %T, got %T", &filterapi.HTTPResponse{}, res)
	}
	assert.IntEQ(303, hres.StatusCode)
}

// TestFilterCallback verifies the authorization server properly redirects when callback path
// is called with correct code and signed state. Verifies that the response from the
// IDP is parsed correctly.
func TestFilterCallback(t *testing.T) {
	setup(t)
	defer teardown(t)

	// 1. We call the the authorization server (filterServer) just to get a signed state token
	// via /authorize redirect.
	res, err := filterClient.Filter(context.Background(), newFilterRequest("GET", "/foo", nil))
	assert.NotError(err)
	assert.Bool(res != nil)

	hres, hresOK := res.(*filterapi.HTTPResponse)
	if !hresOK {
		t.Fatalf("Expected %T, got %T", &filterapi.HTTPResponse{}, res)
	}
	loc := hres.Header.Get("location")
	assert.StrNotEmpty(loc)
	u, err := url.Parse(loc)
	assert.NotError(err)
	assert.IntEQ(551, len(u.Query().Get("state")))

	// 2. Now we call the authorization server (filterServer) again with a signed state and
	// code query params. Note that by calling it with code=authorize, our
	// fake IDP server (idpServer) will respond with a mocked access_token.
	//
	// We check if the request contains the redirect with the original
	// request path `/foo`, client-id and cookie.
	res, err = filterClient.Filter(context.Background(), newFilterRequest("GET", fmt.Sprintf("/callback?state=%s&code=authorize", u.Query().Get("state")), nil))
	assert.NotError(err)
	assert.Bool(res != nil)

	hres, hresOK = res.(*filterapi.HTTPResponse)
	if !hresOK {
		t.Fatalf("Expected %T, got %T", &filterapi.HTTPResponse{}, res)
	}
	assert.IntEQ(307, hres.StatusCode)
	assert.StrEQ("http://lukeshu.com/foo", hres.Header.Get("location"))
	tmp := &http.Response{Header: hres.Header} // make use of net/http's Set-Cookie parser
	cookie := tmp.Cookies()[0]
	assert.StrEQ("access_token", cookie.Name)
	assert.StrEQ("mocked_token_123", cookie.Value)
}

// TestFilterCallback verifies that the authorization server return 401 when code is
// is not present in the request to the callback endpoint.
func TestFilterCallbackNoCode(t *testing.T) {
	setup(t)
	defer teardown(t)

	// 1
	res, err := filterClient.Filter(context.Background(), newFilterRequest("GET", "/foo", nil))
	assert.NotError(err)
	assert.Bool(res != nil)

	hres, hresOK := res.(*filterapi.HTTPResponse)
	if !hresOK {
		t.Fatalf("Expected %T, got %T", &filterapi.HTTPResponse{}, res)
	}
	u, err := url.Parse(hres.Header.Get("location"))
	assert.NotError(err)
	assert.IntEQ(551, len(u.Query().Get("state")))

	// 2
	res, err = filterClient.Filter(context.Background(), newFilterRequest("GET", fmt.Sprintf("/callback?state=%s", u.Query().Get("state")), nil))
	assert.NotError(err)
	assert.Bool(res != nil)

	hres, hresOK = res.(*filterapi.HTTPResponse)
	if !hresOK {
		t.Fatalf("Expected %T, got %T", &filterapi.HTTPResponse{}, res)
	}
	assert.IntEQ(401, hres.StatusCode)
}
