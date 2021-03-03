// +build test
// +build !legacymode

package agent_test

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"github.com/datawire/ambassador/pkg/api/agent"
	snapshotTypes "github.com/datawire/ambassador/pkg/snapshot/v1"
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/url"
	"path/filepath"
	"runtime"
	"strconv"
	"testing"
	"time"
)

type retryable struct {
	log    *bytes.Buffer
	failed bool
}

func (r *retryable) Errorf(s string, v ...interface{}) {
	r.logf(s, v...)
	r.failed = true
}

func (r *retryable) logf(s string, v ...interface{}) {
	fmt.Fprint(r.log, "\n")
	fmt.Fprint(r.log, lineNumber())
	fmt.Fprintf(r.log, s, v...)
}

func lineNumber() string {
	_, file, line, ok := runtime.Caller(3)
	if !ok {
		return ""
	}
	return filepath.Base(file) + ":" + strconv.Itoa(line) + ": "
}

func retry(t *testing.T, numRetries int, f func(r *retryable)) bool {

	var lastLog *bytes.Buffer
	for i := 0; i < numRetries; i++ {
		r := &retryable{log: &bytes.Buffer{}, failed: false}
		f(r)
		if !r.failed {
			return true
		}
		lastLog = r.log
		time.Sleep(time.Second * 5)
	}
	t.Logf("Failed after %d attempts:%s", numRetries, lastLog.String())
	t.Fail()

	return false
}

// this is just to sanity check that the ambassador agent can successfully communicate with its
// server counterpart
// This is basically just testing that, when configured with the yaml we give to clients (or
// something like it) the agent doesn't just completely fall on its face
// Any test that's more complicated should live in apro/cmd/agent/
func TestAgentBasicFunctionality(mt *testing.T) {
	retry(mt, 5, func(t *retryable) {

		mockAgentURL, err := url.Parse("http://agentcom-server.default.svc.cluster.local:3001/lastSnapshot")
		if err != nil {
			t.Errorf("error parsing url: %+v", err)
			return
		}

		client := &http.Client{
			Transport: &http.Transport{
				// #nosec G402
				TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
			},
		}

		resp, err := client.Do(&http.Request{
			Method: "GET",
			URL:    mockAgentURL,
		})
		if err != nil {
			t.Errorf("error making request: %+v", err)
			return
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Errorf("mock server responded with status code %d, wanted 200", resp.StatusCode)
			return
		}
		reportSnapshot := &agent.Snapshot{}
		err = json.NewDecoder(resp.Body).Decode(reportSnapshot)

		if err != nil {
			t.Errorf("Error unmarshalling json: %+v", err)
			return
		}

		// basic sanity assertions. the tests in apro/cmd/agent/ cover these in more detail, so we
		// don't need to here
		assert.NotEmpty(t, reportSnapshot.Identity.ClusterId)
		assert.NotEmpty(t, reportSnapshot.Identity.Hostname)
		assert.NotEmpty(t, reportSnapshot.RawSnapshot)
		assert.NotEmpty(t, reportSnapshot.ApiVersion)
		assert.NotEmpty(t, reportSnapshot.SnapshotTs)
		assert.Equal(t, reportSnapshot.ApiVersion, snapshotTypes.ApiVersion)

		ambSnapshot := &snapshotTypes.Snapshot{}
		err = json.Unmarshal(reportSnapshot.RawSnapshot, ambSnapshot)
		if err != nil {
			t.Errorf("error unmarshalling json: %+v", err)
			return
		}
		assert.NotEmpty(t, ambSnapshot.Kubernetes)

		// pods not being empty basically ensures that the rbac in the yaml is correct
		assert.NotEmpty(t, ambSnapshot.Kubernetes.Pods)

		// just make sure the stuff we really need for the service catalog is in there
		assert.NotEmpty(t, ambSnapshot.Kubernetes.Services)
		assert.NotEmpty(t, ambSnapshot.Kubernetes.Mappings)
	})
}
