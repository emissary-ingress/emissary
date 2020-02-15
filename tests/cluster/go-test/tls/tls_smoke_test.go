// +build test

package tls_test

import (
	"crypto/tls"
	"net/http"
	"os"
	"testing"

	"github.com/datawire/apro/lib/testutil"
)

var testAmbassadorHostname = func() string {
	ret := os.Getenv("TEST_AMBASSADOR_HOSTNAME")
	if ret == "" {
		ret = "ambassador.ambassador.svc.cluster.local"
	}
	return ret
}()

func TestTLSSmoketest(t *testing.T) {
	t.Parallel()

	testcases := map[string]struct {
		URL                string
		ExpectedStatusCode int
		ExpectedHeader     http.Header
	}{
		// Currently, when there are no Hosts or TLSContexts configured, it does *not* redirect cleartext to
		// TLS; instead showing an informative 404 HTML page that explains that when going to `https://` there
		// will be a scary browser warning because no TLS has been configured yet, so it will be using the
		// self-signed fallback certificate.  This may change in the future with `edgectl install`.
		//
		//"cleartext-root":  {"http://" + testAmbassadorHostname + "/", http.StatusMovedPermanently, http.Header{"Location": {"https://" + testAmbassadorHostname + "/"}}},
		"cleartext-root":  {"http://" + testAmbassadorHostname + "/", http.StatusNotFound, nil},
		"encrypted-root":  {"https://" + testAmbassadorHostname + "/", http.StatusNotFound, nil},
		"cleartext-webui": {"http://" + testAmbassadorHostname + "/edge_stack/admin/", http.StatusMovedPermanently, http.Header{"Location": {"https://" + testAmbassadorHostname + "/edge_stack/admin/"}}},
		"encrypted-webui": {"https://" + testAmbassadorHostname + "/edge_stack/admin/", http.StatusOK, http.Header{"Content-Type": {"text/html; charset=utf-8"}}},
		"acme-challenge":  {"http://" + testAmbassadorHostname + "/.well-known/acme-challenge/foobar", http.StatusNotFound, nil},
	}

	client := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
		Transport: &http.Transport{
			// #nosec G402
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}

	for testname, testdata := range testcases {
		testdata := testdata // capture loop variable
		t.Run(testname, func(t *testing.T) {
			t.Parallel()
			assert := &testutil.Assert{T: t}

			resp, err := client.Get(testdata.URL)
			assert.NotError(err)
			defer resp.Body.Close()
			assert.HTTPResponseStatusEQ(resp, testdata.ExpectedStatusCode)
			for key := range testdata.ExpectedHeader {
				if testdata.ExpectedHeader.Get(key) != resp.Header.Get(key) {
					t.Errorf("Wrong header[%q]:\n"+
						"\texpected: %q\n"+
						"\treceived: %q\n",
						key, testdata.ExpectedHeader.Get(key), resp.Header.Get(key))
				}
			}
		})
	}
}
