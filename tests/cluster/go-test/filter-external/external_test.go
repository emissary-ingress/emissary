// +build test

package external_test

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/datawire/apro/lib/testutil"
)

type Body struct {
	Headers map[string]string `json:"headers"`
	Msg     *string           `json:"msg"`
}

func assertHTTPHeaderNotSet(t *testing.T, header http.Header, key string) {
	t.Helper()
	val, set := header[http.CanonicalHeaderKey(key)]
	if set {
		t.Fatalf("Expected HTTP Header %q to not be set, but got %#v", http.CanonicalHeaderKey(key), val)
	}
}

func assertJSONHeaderNotSet(t *testing.T, header map[string]string, key string) {
	t.Helper()
	val, set := header[http.CanonicalHeaderKey(key)]
	if set {
		t.Fatalf("Expected JSON Header %q to not be set, but got %q", http.CanonicalHeaderKey(key), val)
	}
}

func assertHTTPHeaderEq(t *testing.T, header http.Header, key string, exval []string) {
	t.Helper()
	val, set := header[http.CanonicalHeaderKey(key)]
	if !set {
		t.Fatalf("Expected HTTP Header %q to be %#v, but was not set", http.CanonicalHeaderKey(key), exval)
	}
	if fmt.Sprintf("%#v", val) != fmt.Sprintf("%#v", exval) {
		t.Fatalf("Expected HTTP Header %q to be %#v, but got %#v", http.CanonicalHeaderKey(key), exval, val)
	}
}

func assertJSONHeaderEq(t *testing.T, header map[string]string, key string, exval string) {
	t.Helper()
	val, set := header[http.CanonicalHeaderKey(key)]
	if !set {
		t.Fatalf("Expected JSON Header %q to be %#v, but was not set", http.CanonicalHeaderKey(key), exval)
	}
	if val != exval {
		t.Fatalf("Expected JSON Header %q to be %q, but got %q", http.CanonicalHeaderKey(key), exval, val)
	}
}

func doRequest(t *testing.T, urlStr string) (*http.Response, Body) {
	t.Helper()
	assert := &testutil.Assert{T: t}

	u, err := url.Parse(urlStr)
	assert.NotError(err)

	client := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
		Transport: &http.Transport{
			// #nosec G402
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}

	res, err := client.Do(&http.Request{
		Method: "GET",
		URL:    u,
		Header: http.Header{
			"X-Allowed-Input-Header":    {"foo"},
			"X-Disallowed-Input-Header": {"bar"},
		},
	})
	assert.NotError(err)
	assert.Bool(res != nil)
	defer res.Body.Close()
	bodyBytes, err := ioutil.ReadAll(res.Body)
	assert.NotError(err)
	t.Logf("Body: %s", bodyBytes)
	var body Body
	assert.NotError(json.Unmarshal(bodyBytes, &body))
	return res, body
}

func inArray(needle string, haystack []string) bool {
	for _, straw := range haystack {
		if needle == straw {
			return true
		}
	}
	return false
}

func TestHTTPExternalModify(t *testing.T) {
	t.Parallel()
	assert := &testutil.Assert{T: t}
	res, body := doRequest(t, "https://ambassador.default.svc.cluster.local/external-http/headers") //nolint:bodyclose
	// HTTP/1.1 200 OK
	// access-control-allow-credentials: true
	// access-control-allow-origin: *
	// content-type: application/json
	// date: Thu, 14 Mar 2019 14:40:12 GMT
	// server: envoy
	// content-length: 470
	// x-envoy-upstream-service-time: 2
	//
	// {
	//   "headers": {
	//     "Accept": "*/*",
	//     "Host": "httpbin.org",
	//     "User-Agent": "curl/7.64.0",
	//     "X-Allowed-Input-Header": "foo",
	//     "X-Allowed-Output-Header": "baz",
	//     "X-Disallowed-Input-Header": "bar",
	//     "X-Envoy-Expected-Rq-Timeout-Ms": "5000",
	//     "X-Envoy-Internal": "true",
	//     "X-Envoy-Original-Path": "/external-http/headers",
	//     "X-Input-Headers": "Accept-Encoding,User-Agent,X-Allowed-Input-Header,X-Forwarded-For,X-Forwarded-Proto"
	//   }
	// }

	assert.Bool(body.Msg == nil)
	assert.Bool(body.Headers != nil)

	assert.HTTPResponseStatusEQ(res, http.StatusOK)
	assertHTTPHeaderNotSet(t, res.Header, "x-allowed-input-header")
	assertHTTPHeaderNotSet(t, res.Header, "x-disallowed-input-header")
	assertHTTPHeaderNotSet(t, res.Header, "x-allowed-output-header")
	assertHTTPHeaderNotSet(t, res.Header, "x-disallowed-output-header")
	assertJSONHeaderEq(t, body.Headers, "x-allowed-input-header", "foo")
	assertJSONHeaderEq(t, body.Headers, "x-disallowed-input-header", "bar")
	assertJSONHeaderEq(t, body.Headers, "x-allowed-output-header", "baz")
	assertJSONHeaderNotSet(t, body.Headers, "x-disallowed-output-header")
	inputHeadersStr, set := body.Headers["X-Input-Headers"]
	assert.Bool(set)
	inputHeaders := strings.Split(inputHeadersStr, ",")
	assert.Bool(inArray("X-Allowed-Input-Header", inputHeaders))
	assert.Bool(!inArray("X-Disallowed-Input-Header", inputHeaders))
}

func TestHTTPExternalIntercept(t *testing.T) {
	t.Parallel()
	assert := &testutil.Assert{T: t}
	res, body := doRequest(t, "https://ambassador.default.svc.cluster.local/external-http/ip") //nolint:bodyclose
	// HTTP/1.1 404 Not Found
	// x-allowed-output-header: baz
	// x-disallowed-output-header: qux
	// date: Thu, 14 Mar 2019 14:45:43 GMT
	// content-length: 22
	// content-type: application/json
	// server: envoy
	//
	// {"msg": "intercepted"}

	assert.Bool(body.Msg != nil)
	assert.Bool(body.Headers == nil)

	assert.HTTPResponseStatusEQ(res, http.StatusNotFound)
	assertHTTPHeaderEq(t, res.Header, "x-allowed-output-header", []string{"baz"})
	assertHTTPHeaderEq(t, res.Header, "x-disallowed-output-header", []string{"qux"})
	assert.StrEQ(*body.Msg, "intercepted")
}

func TestHTTPExternalInterceptWithRedirect(t *testing.T) {
	t.Parallel()
	assert := &testutil.Assert{T: t}
	res, body := doRequest(t, "https://ambassador.default.svc.cluster.local/external-http/redirect") //nolint:bodyclose
	// HTTP/1.1 302 Found
	// location: https://example.com/
	// date: Fri, 06 Sep 2019 13:37:02 GMT
	// content-length: 21
	// content-type: application/json
	// server: envoy
	//
	// {"msg": "redirected"}

	assert.Bool(body.Msg != nil)
	assert.Bool(body.Headers == nil)

	assert.HTTPResponseStatusEQ(res, http.StatusFound)
	assertHTTPHeaderEq(t, res.Header, "location", []string{"https://example.com/"})
}

func TestGRPCExternalModify(t *testing.T) {
	t.Parallel()
	assert := &testutil.Assert{T: t}
	res, body := doRequest(t, "https://ambassador.default.svc.cluster.local/external-grpc/headers") //nolint:bodyclose
	// HTTP/1.1 200 OK
	// access-control-allow-credentials: true
	// access-control-allow-origin: *
	// content-type: application/json
	// date: Thu, 14 Mar 2019 14:48:10 GMT
	// server: envoy
	// content-length: 320
	// x-envoy-upstream-service-time: 2
	//
	// {
	//   "headers": {
	//     "Accept": "*/*",
	//     "Host": "httpbin.org",
	//     "User-Agent": "curl/7.64.0",
	//     "X-Allowed-Input-Header": "foo",
	//     "X-Disallowed-Input-Header": "bar",
	//     "X-Envoy-Expected-Rq-Timeout-Ms": "5000",
	//     "X-Envoy-Internal": "true",
	//     "X-Envoy-Original-Path": "/external-grpc/headers"
	//   }
	// }

	assert.Bool(body.Msg == nil)
	assert.Bool(body.Headers != nil)

	assert.HTTPResponseStatusEQ(res, http.StatusOK)
	assertHTTPHeaderNotSet(t, res.Header, "x-allowed-input-header")
	assertHTTPHeaderNotSet(t, res.Header, "x-disallowed-input-header")
	assertHTTPHeaderNotSet(t, res.Header, "x-allowed-output-header")
	assertHTTPHeaderNotSet(t, res.Header, "x-disallowed-output-header")
	assertJSONHeaderEq(t, body.Headers, "x-input-x-allowed-input-header", "foo")
	assertJSONHeaderEq(t, body.Headers, "x-input-x-disallowed-input-header", "bar")
	assertJSONHeaderEq(t, body.Headers, "x-allowed-input-header", "foo,after") // append
	assertJSONHeaderEq(t, body.Headers, "x-disallowed-input-header", "after")  // override
	//assertJSONHeaderEq(t, body.Headers, "x-allowed-output-header", "baz") // append https://github.com/datawire/ambassador/issues/1313
	assertJSONHeaderEq(t, body.Headers, "x-disallowed-output-header", "qux") // override
	assertJSONHeaderEq(t, body.Headers, "x-input-x-allowed-input-header", "foo")
	assertJSONHeaderEq(t, body.Headers, "x-input-x-disallowed-input-header", "bar")
}

func TestGRPCExternalIntercept(t *testing.T) {
	t.Parallel()
	assert := &testutil.Assert{T: t}
	res, body := doRequest(t, "https://ambassador.default.svc.cluster.local/external-grpc/ip") //nolint:bodyclose
	// HTTP/1.1 200 OK
	// content-length: 22
	// content-type: application/json
	// date: Thu, 14 Mar 2019 15:30:34 GMT
	// server: envoy
	//
	// {"msg": "intercepted"}

	assert.Bool(body.Msg != nil)
	assert.Bool(body.Headers == nil)

	assert.HTTPResponseStatusEQ(res, http.StatusOK)
	assertHTTPHeaderEq(t, res.Header, "x-allowed-output-header", []string{"baz"})
	assertHTTPHeaderEq(t, res.Header, "x-disallowed-output-header", []string{"qux"})
	assert.StrEQ(*body.Msg, "intercepted")
}
