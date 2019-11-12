// +build test

package actions_test

import (
	"crypto/tls"
	"net/http"
	"net/url"
	"testing"

	"github.com/dgrijalva/jwt-go"

	"github.com/datawire/apro/lib/testutil"
)

func urlMust(u *url.URL, e error) *url.URL {
	if e != nil {
		panic(e)
	}
	return u
}

type testcase struct {
	URL            *url.URL
	Header         http.Header
	ExpectedStatus int
}

func (tc testcase) Run(t *testing.T) {
	assert := &testutil.Assert{T: t}

	client := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
		Transport: &http.Transport{
			// #nosec G402
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}

	resp, err := client.Do(&http.Request{
		Method: "GET",
		URL:    tc.URL,
		Header: tc.Header,
	})
	assert.NotError(err)
	assert.Bool(resp != nil)
	defer resp.Body.Close()

	assert.HTTPResponseStatusEQ(resp, tc.ExpectedStatus)
}

func testJWT(t *testing.T) string {
	assert := &testutil.Assert{T: t}

	tokenStruct := jwt.NewWithClaims(jwt.GetSigningMethod("none"), jwt.MapClaims{
		"sub":  "1234567890",
		"name": "John Doe",
		"iat":  1516239022,
	})
	tokenString, err := tokenStruct.SignedString(jwt.UnsafeAllowNoneSignatureType)
	assert.NotError(err)

	return tokenString
}

func TestIfRequestHeader(t *testing.T) {
	u := urlMust(url.Parse("https://ambassador.ambassador.svc.cluster.local/filter-actions/if/headers"))
	jwt := testJWT(t)

	testcases := map[string]testcase{
		"noJWT":      {URL: u, Header: nil, ExpectedStatus: http.StatusSeeOther},                                                // try OAuth2
		"invalidJWT": {URL: u, Header: http.Header{"Authorization": {"Bearer bogus"}}, ExpectedStatus: http.StatusUnauthorized}, // forbid
		"validJWT":   {URL: u, Header: http.Header{"Authorization": {"Bearer " + jwt}}, ExpectedStatus: http.StatusOK},          // allow
	}

	for tcName, tc := range testcases {
		tc := tc // capture loop variable
		t.Run(tcName, tc.Run)
	}
}

func TestOnAction(t *testing.T) {
	u := urlMust(url.Parse("https://ambassador.ambassador.svc.cluster.local/filter-actions/on/headers"))
	jwt := testJWT(t)

	testcases := map[string]testcase{
		"noJWT":      {URL: u, Header: nil, ExpectedStatus: http.StatusSeeOther},                                            // try OAuth2
		"invalidJWT": {URL: u, Header: http.Header{"Authorization": {"Bearer bogus"}}, ExpectedStatus: http.StatusSeeOther}, // try OAuth2
		"validJWT":   {URL: u, Header: http.Header{"Authorization": {"Bearer " + jwt}}, ExpectedStatus: http.StatusOK},      // allow
	}

	for tcName, tc := range testcases {
		tc := tc // capture loop variable
		t.Run(tcName, tc.Run)
	}
}
