// +build test

package jwt_test

import (
	"crypto/tls"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"testing"

	jwt "github.com/dgrijalva/jwt-go"

	"github.com/datawire/apro/lib/testutil"
)

type TestHeader struct {
	Template string
	Expect   string
}

func TestJWTInjectHeaders(t *testing.T) {
	t.Parallel()
	assert := &testutil.Assert{T: t}

	// build the test-case /////////////////////////////////////////////////
	tokenStruct := jwt.NewWithClaims(jwt.GetSigningMethod("none"), jwt.MapClaims{
		"sub":  "1234567890",
		"name": "John Doe",
		"iat":  1516239022,
	})
	tokenStruct.Header["extra"] = "so much"
	tokenString, err := tokenStruct.SignedString(jwt.UnsafeAllowNoneSignatureType)
	assert.NotError(err)

	testHeaders := map[string]string{
		"X-Fixed-String":        "Fixed String",
		"X-Override":            "after",
		"X-Token-String":        tokenString,
		"X-Token-H-Alg":         "none",
		"X-Token-H-Typ":         "JWT",
		"X-Token-H-Extra":       "so much",
		"X-Token-C-Sub":         "1234567890",
		"X-Token-C-Name":        "John Doe",
		"X-Token-C-Iat":         "1.516239022e+09",
		"X-Token-C-Iat-Decimal": "1516239022",
		"X-Token-S":             tokenString[strings.LastIndexByte(tokenString, '.')+1:],
		"X-Authorization":       `Authenticated JWT; sub=1234567890; name="John Doe"`,
		"X-UA":                  "Go-http-client/1.1",
	}

	// run the filter //////////////////////////////////////////////////////
	u, err := url.Parse("https://ambassador.standalone.svc.cluster.local/jwt/headers")
	assert.NotError(err)
	client := &http.Client{
		Transport: &http.Transport{
			// #nosec G402
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}
	resp, err := client.Do(&http.Request{
		Method: "GET",
		URL:    u,
		Header: http.Header{
			"Authorization": {"Bearer " + tokenString},
			"X-Override":    {"before"},
		},
	})
	assert.NotError(err)
	assert.Bool(resp != nil)
	defer resp.Body.Close()

	// inspect the result //////////////////////////////////////////////////

	assert.HTTPResponseStatusEQ(resp, http.StatusOK)
	bodyBytes, err := ioutil.ReadAll(resp.Body)
	assert.NotError(err)
	t.Logf("Body: %s", bodyBytes)
	var body struct {
		Headers map[string]string `json:"headers"`
	}
	assert.NotError(json.Unmarshal(bodyBytes, &body))

	header := make(http.Header)
	for key, val := range body.Headers {
		header.Set(key, val)
	}

	for key, val := range testHeaders {
		if val != header.Get(key) {
			t.Errorf("Wrong header[%q]:\n"+
				"\texpected: %q\n"+
				"\treceived: %q\n",
				key, val, header.Get(key))
		}
	}
}

// customErrorResponse is the custom bodyTemplate in 04-filter-jwt.yaml
type customErrorResponse struct {
	ErrorMessage    string `json:"errorMessage"`
	AltErrorMessage string `json:"altErrorMessage"`
	ErrorCode       int    `json:"errorCode"`
	HTTPStatus      string `json:"httpStatus"`
	//RequestID       string `json:"requestId"`
}

func TestJWTErrorResponse(t *testing.T) {
	t.Parallel()
	assert := &testutil.Assert{T: t}

	// build the test-case /////////////////////////////////////////////////
	tokenString, err := jwt.NewWithClaims(jwt.GetSigningMethod("none"), jwt.MapClaims{
		"sub":  "1234567890",
		"name": "John Doe",
		"iat":  1516239022,
		"exp":  1516239023,
	}).SignedString(jwt.UnsafeAllowNoneSignatureType)
	assert.NotError(err)

	testHeaders := map[string]string{
		"X-Correlation-ID": "foobar",
		"Content-Type":     "application/json",
	}
	testBody := customErrorResponse{
		ErrorMessage:    "Token validation error: token is invalid: errorFlags=0x00000010=(ValidationErrorExpired) wrappedError=(Token is expired)",
		AltErrorMessage: "expired",
		ErrorCode:       16,
		HTTPStatus:      "401",
		//"requestId": "7167523368329307446"
	}

	// run the filter //////////////////////////////////////////////////////
	u, err := url.Parse("https://ambassador.standalone.svc.cluster.local/jwt/headers")
	assert.NotError(err)
	client := &http.Client{
		Transport: &http.Transport{
			// #nosec G402
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}
	resp, err := client.Do(&http.Request{
		Method: "GET",
		URL:    u,
		Header: http.Header{
			"Authorization":    {"Bearer " + tokenString},
			"X-Override":       {"before"},
			"X-Correlation-ID": {"foobar"},
		},
	})
	assert.NotError(err)
	assert.Bool(resp != nil)
	defer resp.Body.Close()

	// inspect the result //////////////////////////////////////////////////

	assert.HTTPResponseStatusEQ(resp, http.StatusUnauthorized)
	bodyBytes, err := ioutil.ReadAll(resp.Body)
	assert.NotError(err)
	t.Logf("Body: %s", bodyBytes)
	var body customErrorResponse
	assert.NotError(json.Unmarshal(bodyBytes, &body))

	for key, val := range testHeaders {
		if val != resp.Header.Get(key) {
			t.Errorf("Wrong header[%q]:\n"+
				"\texpected: %q\n"+
				"\treceived: %q\n",
				key, val, resp.Header.Get(key))
		}
	}

	if testBody != body {
		t.Errorf("Wrong body:\n"+
			"\texpected: %#v\n"+
			"\treceived: %#v\n",
			testBody, body)
	}
}
