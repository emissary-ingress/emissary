// +build test
// +build !legacymode

package agent_test

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"testing"
	"time"

	"github.com/datawire/ambassador/pkg/api/agent"
	"github.com/datawire/ambassador/pkg/k8s"
	"github.com/datawire/ambassador/pkg/kubeapply"
	snapshotTypes "github.com/datawire/ambassador/pkg/snapshot/v1"
	"github.com/datawire/dlib/dexec"
	"github.com/stretchr/testify/assert"
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
	kubeconfig := os.Getenv("DEV_KUBECONFIG")
	if kubeconfig == "" {
		mt.Fatalf("DEV_KUBECONFIG must be set")
	}
	defer deleteArgoResources(mt, kubeconfig)
	retry(mt, 5, func(t *retryable) {
		reportSnapshot, ambSnapshot := getAgentComSnapshots(t, false)

		assert.NotNil(t, reportSnapshot)
		assert.NotNil(t, ambSnapshot)
		if reportSnapshot == nil || ambSnapshot == nil {
			return
		}
		// basic sanity assertions. the tests in apro/cmd/agent/ cover these in more detail, so we
		// don't need to here
		assert.NotNil(t, reportSnapshot.Identity)
		if reportSnapshot.Identity == nil {
			return
		}

		assert.NotEmpty(t, reportSnapshot.Identity.ClusterId)
		assert.NotEmpty(t, reportSnapshot.Identity.Hostname)
		assert.NotEmpty(t, reportSnapshot.RawSnapshot)
		assert.NotEmpty(t, reportSnapshot.ApiVersion)
		assert.NotEmpty(t, reportSnapshot.SnapshotTs)
		assert.Equal(t, reportSnapshot.ApiVersion, snapshotTypes.ApiVersion)

		assert.NotEmpty(t, ambSnapshot.Kubernetes)

		// pods not being empty basically ensures that the rbac in the yaml is correct
		assert.NotEmpty(t, ambSnapshot.Kubernetes.Pods)

		// just make sure the stuff we really need for the service catalog is in there
		assert.NotEmpty(t, ambSnapshot.Kubernetes.Services)
		assert.NotEmpty(t, ambSnapshot.Kubernetes.Mappings)

		applyArgoResources(t, kubeconfig)
		reportSnapshot, ambSnapshot = getAgentComSnapshots(t, true)

		assert.NotNil(t, reportSnapshot)
		assert.NotNil(t, ambSnapshot)
		if reportSnapshot == nil || ambSnapshot == nil {
			return
		}
		// basic sanity assertions. the tests in apro/cmd/agent/ cover these in more detail, so we
		// don't need to here
		assert.NotNil(t, reportSnapshot.Identity)
		if reportSnapshot.Identity == nil {
			return
		}
		assert.NotEmpty(t, ambSnapshot.Kubernetes.ArgoRollouts, "No argo rollouts found in snapshot")
		assert.NotEmpty(t, ambSnapshot.Kubernetes.ArgoApplications, "No argo applications found in snapshot")
	})
}

func getAgentComSnapshots(t *retryable, waitArgo bool) (*agent.Snapshot, *snapshotTypes.Snapshot) {
	found := false
	reportSnapshot := &agent.Snapshot{}
	ambSnapshot := &snapshotTypes.Snapshot{}
	mockAgentURL, err := url.Parse("http://agentcom-server.default.svc.cluster.local:3001/lastSnapshot")
	if err != nil {
		t.Errorf("error parsing url: %+v", err)
		return nil, nil
	}

	// now we're going to go copy the snapshot.json file from our fake agentcom
	// when the agentcom gets a snapshot from the agent, it'll store it at /tmp/snapshot.json
	// we do this in a loop because it might take ambassador and the agent a sec to get into the
	// state we're asserting. this is okay, this test is just to make sure that the agent RBAC
	// is correct and that the agent can talk to the ambassador-agent service.
	// any tests that do any more complicated assertions should live in ambassador.git/pkg/agent
	for i := 0; i < 30; i++ {

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
			t.logf("error making request: %+v", err)
			continue
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.logf("mock server responded with status code %d, wanted 200", resp.StatusCode)
			continue
		}
		found = true
		body, err := ioutil.ReadAll(resp.Body)
		err = json.Unmarshal(body, reportSnapshot)

		if err != nil {
			t.Errorf("Could not unmarshal report snapshot")
			return nil, nil
		}

		err = json.Unmarshal(reportSnapshot.RawSnapshot, ambSnapshot)
		if err != nil {
			t.Errorf("Could not unmarshal ambassador snapshot")
			return nil, nil
		}
		if !snapshotIsSane(ambSnapshot, t, waitArgo) {
			continue
		}
		break
	}
	assert.True(t, found, "Could not cp file from agentcom")
	return reportSnapshot, ambSnapshot
}

func snapshotIsSane(ambSnapshot *snapshotTypes.Snapshot, t *retryable, hasArgo bool) bool {
	if ambSnapshot.Kubernetes == nil {
		t.logf("K8s snapshot empty, retrying")
		return false
	}
	if len(ambSnapshot.Kubernetes.Services) == 0 {
		t.logf("K8s snapshot services empty, retrying")
		return false
	}
	if len(ambSnapshot.Kubernetes.Mappings) == 0 {
		t.logf("K8s snapshot mappings empty, retrying")
		return false
	}
	if len(ambSnapshot.Kubernetes.Pods) == 0 {
		t.logf("K8s snapshot pods empty, retrying")
		return false
	}
	if hasArgo && len(ambSnapshot.Kubernetes.ArgoRollouts) == 0 {
		t.logf("K8s snapshot argo rollouts empty, retrying")
		return false
	}
	if hasArgo && len(ambSnapshot.Kubernetes.ArgoApplications) == 0 {
		t.logf("K8s snapshot argo applications empty, retrying")
		return false
	}
	if !hasArgo && len(ambSnapshot.Kubernetes.ArgoRollouts) != 0 {
		t.logf("K8s snapshot argo rollouts should be empty, retrying")
		return false
	}
	if !hasArgo && len(ambSnapshot.Kubernetes.ArgoApplications) != 0 {
		t.logf("K8s snapshot argo applications should be empty, retrying")
		return false
	}

	return true
}

func applyArgoResources(t *retryable, kubeconfig string) {
	kubeinfo := k8s.NewKubeInfo(kubeconfig, "", "")
	err := kubeapply.Kubeapply(kubeinfo, time.Minute, true, false, "./test/argo-rollouts-crd.yaml")
	assert.Nil(t, err)
	err = kubeapply.Kubeapply(kubeinfo, time.Minute, true, false, "./test/argo-rollouts.yaml")
	assert.Nil(t, err)
	err = kubeapply.Kubeapply(kubeinfo, time.Minute, true, false, "./test/argo-application-crd.yaml")
	assert.Nil(t, err)
	err = kubeapply.Kubeapply(kubeinfo, time.Minute, true, false, "./test/argo-application.yaml")
	assert.Nil(t, err)
}

func deleteArgoResources(t *testing.T, kubeconfig string) {
	files := []string{"./test/argo-rollouts-crd.yaml", "./test/argo-rollouts.yaml", "./test/argo-application-crd.yaml", "./test/argo-application.yaml"}
	ctx := context.Background()
	for _, f := range files {
		cmd := dexec.CommandContext(ctx, "kubectl", "--kubeconfig", kubeconfig, "delete", "--ignore-not-found=true", "-f", f)
		out, err := cmd.CombinedOutput()
		t.Log(fmt.Sprintf("Kubectl delete output: %s", out))
		if err != nil {
			t.Logf("Error running kubectl delete crd rollouts: %s", err)
		}

	}
}
