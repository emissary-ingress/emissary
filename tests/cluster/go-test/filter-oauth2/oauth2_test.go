// +build test

package oauth2_test

import (
	"crypto/tls"
	"net/http"
	"net/url"
	"testing"

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
	urlAny := urlMust(url.Parse("https://ambassador.default.svc.cluster.local/auth0/httpbin/headers"))
	urlXHR := urlMust(url.Parse("https://ambassador.default.svc.cluster.local/auth0-k8s/httpbin/headers"))

	testcases := map[string]testcase{
		"anyNone":  {URL: urlAny, Header: nil, ExpectedStatus: http.StatusSeeOther},
		"anyEmpty": {URL: urlAny, Header: http.Header{"X-Requested-With": {""}}, ExpectedStatus: http.StatusSeeOther},
		"anyXHR":   {URL: urlAny, Header: http.Header{"X-Requested-With": {"XMLHttpRequest"}}, ExpectedStatus: http.StatusForbidden},
		"anyOther": {URL: urlAny, Header: http.Header{"X-Requested-With": {"frob"}}, ExpectedStatus: http.StatusForbidden},

		"xhrNone":  {URL: urlXHR, Header: nil, ExpectedStatus: http.StatusSeeOther},
		"xhrEmpty": {URL: urlXHR, Header: http.Header{"X-Requested-With": {""}}, ExpectedStatus: http.StatusSeeOther},
		"xhrXHR":   {URL: urlXHR, Header: http.Header{"X-Requested-With": {"XMLHttpRequest"}}, ExpectedStatus: http.StatusUnauthorized},
		"xhrOther": {URL: urlXHR, Header: http.Header{"X-Requested-With": {"frob"}}, ExpectedStatus: http.StatusSeeOther},
	}

	for tcName, tc := range testcases {
		tc := tc // capture loop variable
		t.Run(tcName, tc.Run)
	}
}
