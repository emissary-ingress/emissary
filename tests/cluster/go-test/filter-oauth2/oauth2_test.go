// +build test

package oauth2_test

import (
	"crypto/tls"
	"net/http"
	"net/url"
	"testing"

	jwt "github.com/dgrijalva/jwt-go"

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
	t.Parallel()
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

func TestInsteadOfRedirect(t *testing.T) {
	t.Parallel()
	assert := &testutil.Assert{T: t}

	urlAny := urlMust(url.Parse("https://ambassador.standalone.svc.cluster.local/auth0/httpbin/headers"))
	urlXHR := urlMust(url.Parse("https://ambassador.standalone.svc.cluster.local/auth0-k8s/httpbin/headers"))
	urlJWT := urlMust(url.Parse("https://ambassador.standalone.svc.cluster.local/auth0-or-jwt/headers"))

	insufficientToken, err := jwt.NewWithClaims(jwt.GetSigningMethod("none"), jwt.MapClaims{
		"sub":   "1234567890",
		"name":  "John Doe",
		"iat":   1516239022,
		"exp":   1616239022,
		"scope": "",
	}).SignedString(jwt.UnsafeAllowNoneSignatureType)
	assert.NotError(err)

	validToken, err := jwt.NewWithClaims(jwt.GetSigningMethod("none"), jwt.MapClaims{
		"sub":   "1234567890",
		"name":  "John Doe",
		"iat":   1516239022,
		"exp":   1616239022,
		"scope": "openid",
	}).SignedString(jwt.UnsafeAllowNoneSignatureType)
	assert.NotError(err)

	testcases := map[string]testcase{
		"anyNone":  {URL: urlAny, Header: nil, ExpectedStatus: http.StatusSeeOther},
		"anyEmpty": {URL: urlAny, Header: http.Header{"X-Requested-With": {""}}, ExpectedStatus: http.StatusSeeOther},
		"anyXHR":   {URL: urlAny, Header: http.Header{"X-Requested-With": {"XMLHttpRequest"}}, ExpectedStatus: http.StatusForbidden},
		"anyOther": {URL: urlAny, Header: http.Header{"X-Requested-With": {"frob"}}, ExpectedStatus: http.StatusForbidden},

		"xhrNone":  {URL: urlXHR, Header: nil, ExpectedStatus: http.StatusSeeOther},
		"xhrEmpty": {URL: urlXHR, Header: http.Header{"X-Requested-With": {""}}, ExpectedStatus: http.StatusSeeOther},
		"xhrXHR":   {URL: urlXHR, Header: http.Header{"X-Requested-With": {"XMLHttpRequest"}}, ExpectedStatus: http.StatusUnauthorized},
		"xhrOther": {URL: urlXHR, Header: http.Header{"X-Requested-With": {"frob"}}, ExpectedStatus: http.StatusSeeOther},

		"jwtNone":         {URL: urlJWT, Header: nil, ExpectedStatus: http.StatusSeeOther},
		"jwtEmpty":        {URL: urlJWT, Header: http.Header{"X-Requested-With": {""}}, ExpectedStatus: http.StatusSeeOther},
		"jwtXHR":          {URL: urlJWT, Header: http.Header{"X-Requested-With": {"XMLHttpRequest"}}, ExpectedStatus: http.StatusUnauthorized},
		"jwtOther":        {URL: urlJWT, Header: http.Header{"X-Requested-With": {"frob"}}, ExpectedStatus: http.StatusUnauthorized},
		"jwtValid":        {URL: urlJWT, Header: http.Header{"X-Requested-With": {"frob"}, "Authorization": {"Bearer " + validToken}}, ExpectedStatus: http.StatusOK},
		"jwtInsufficient": {URL: urlJWT, Header: http.Header{"X-Requested-With": {"frob"}, "Authorization": {"Bearer " + insufficientToken}}, ExpectedStatus: http.StatusForbidden},
	}

	for tcName, tc := range testcases {
		tc := tc // capture loop variable
		t.Run(tcName, tc.Run)
	}
}

func TestClientCredentials(t *testing.T) {
	u := urlMust(url.Parse("https://ambassador.standalone.svc.cluster.local/okta-client-credentials/httpbin/headers"))

	testcases := map[string]testcase{
		"empty":   {URL: u, Header: nil, ExpectedStatus: http.StatusForbidden},
		"valid":   {URL: u, Header: http.Header{"X-Ambassador-Client-ID": {"0oa1seewd25KEdjRk357"}, "X-Ambassador-Client-Secret": {"suMFuqElbCFBhVw760Nf-TeuBLjR7uoUWpANM8bS"}}, ExpectedStatus: http.StatusOK},
		"invalid": {URL: u, Header: http.Header{"X-Ambassador-Client-ID": {"0oa1seewd25KEdjRk357"}, "X-Ambassador-Client-Secret": {"bogus"}}, ExpectedStatus: http.StatusForbidden},
	}

	for tcName, tc := range testcases {
		tc := tc // capture loop variable
		t.Run(tcName, tc.Run)
	}
}
