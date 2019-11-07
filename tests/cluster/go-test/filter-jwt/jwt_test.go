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
