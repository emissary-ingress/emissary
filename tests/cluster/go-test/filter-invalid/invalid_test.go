// +build test

package invalid_test

import (
	"crypto/tls"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/datawire/apro/lib/testutil"
)

func TestInvalid(t *testing.T) {
	t.Parallel()
	assert := &testutil.Assert{T: t}

	u, err := url.Parse("https://ambassador.ambassador.svc.cluster.local/invalid/headers")
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
	})
	assert.NotError(err)
	assert.Bool(resp != nil)
	defer resp.Body.Close()

	// The 04-filter-invalid.yaml says `jwksURI: not-a-uri`, so we
	// should expect an HTTP 500 error complaining about that.
	assert.HTTPResponseStatusEQ(resp, http.StatusInternalServerError)
	bodyBytes, err := ioutil.ReadAll(resp.Body)
	assert.NotError(err)
	t.Logf("Body: %s", bodyBytes)
	assert.Bool(strings.Contains(string(bodyBytes), `jwksURI is not an absolute URI`))
}
