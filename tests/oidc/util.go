package oidc

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"
	"time"
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
		if pos, contained := contains(cookie.Name, names); contained {
			result[cookie.Name] = cookie.Value
			remove(pos, names)
		}
	}

	//if len(names) != 0 {
	//	err = fmt.Errorf("not all cookies found: %v\n", names)
	//}

	return
}

func contains(value string, items []string) (int, bool) {
	for idx, item := range items {
		if item == value {
			return idx, true
		}
	}

	return -1, false
}

func remove(pos int, src []string) []string {
	src[len(src)-1], src[pos] = src[pos], src[len(src)-1]
	return src[:len(src)-1]
}

// returns an http client that is configured for use in OpenID Connect authentication tests. The client is configured
// to ignore self-signed TLS certificates and to not follow redirects automatically.
func newHTTPClient(timeout time.Duration, enableCookies bool) http.Client {
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

func createHTTPRequest(method string, url url.URL) (*http.Request, error) {
	request, err := http.NewRequest(method, url.String(), nil)
	if err != nil {
		return nil, err
	}

	return request, nil
}
