package entrypoint

import (
	"bytes"
	"context"
	"io/ioutil"
	"net/http"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

var sanitizeExternalSnapshotTests = []struct {
	testName                 string
	rawJSON                  string
	hasEdgeStackSidecar      bool
	edgeStackSidecarResponse int
	expectedSanitizedJSON    string
}{
	{
		testName:                 "no AmbassadorMeta",
		rawJSON:                  `{"AmbassadorMeta":null,"Kubernetes":null,"Consul":null,"Deltas":null,"Invalid":null}`,
		hasEdgeStackSidecar:      true,
		edgeStackSidecarResponse: 200,
		expectedSanitizedJSON:    `{"AmbassadorMeta":null,"Kubernetes":null,"Consul":null,"Deltas":null,"Invalid":null}`,
	},
	{
		testName:                 "AmbassadorMeta with sidecar",
		rawJSON:                  `{"AmbassadorMeta":{"cluster_id":"","ambassador_id":"","ambassador_version":"","kube_version":""}}`,
		hasEdgeStackSidecar:      true,
		edgeStackSidecarResponse: 200,
		expectedSanitizedJSON:    `{"AmbassadorMeta":{"cluster_id":"","ambassador_id":"","ambassador_version":"","kube_version":"","sidecar":{"contains":"raw sidecar process-info content"}},"Kubernetes":null,"Consul":null,"Deltas":null,"Invalid":null}`,
	},
	{
		testName:                 "AmbassadorMeta with bad sidecar response",
		rawJSON:                  `{"AmbassadorMeta":{"cluster_id":"","ambassador_id":"","ambassador_version":"","kube_version":""}}`,
		hasEdgeStackSidecar:      true,
		edgeStackSidecarResponse: 500,
		expectedSanitizedJSON:    `{"AmbassadorMeta":{"cluster_id":"","ambassador_id":"","ambassador_version":"","kube_version":"","sidecar":null},"Kubernetes":null,"Consul":null,"Deltas":null,"Invalid":null}`,
	},
	{
		testName:              "AmbassadorMeta without any sidecar",
		rawJSON:               `{"AmbassadorMeta":{"cluster_id":"","ambassador_id":"","ambassador_version":"","kube_version":""}}`,
		hasEdgeStackSidecar:   false,
		expectedSanitizedJSON: `{"AmbassadorMeta":{"cluster_id":"","ambassador_id":"","ambassador_version":"","kube_version":"","sidecar":null},"Kubernetes":null,"Consul":null,"Deltas":null,"Invalid":null}`,
	},
}

func TestSanitizeExternalSnapshot(t *testing.T) {
	const isEdgeStackEnvironmentVariable = "EDGE_STACK"
	const rawSidecarProcessInfoResponse = `{"contains":"raw sidecar process-info content"}`

	for _, sanitizeExternalSnapshotTest := range sanitizeExternalSnapshotTests {
		t.Run(sanitizeExternalSnapshotTest.testName, func(innerT *testing.T) {
			defer os.Unsetenv(isEdgeStackEnvironmentVariable)
			if sanitizeExternalSnapshotTest.hasEdgeStackSidecar {
				os.Setenv(isEdgeStackEnvironmentVariable, "true")
			}

			client := newHTTPTestClient(func(req *http.Request) *http.Response {
				assert.Equal(t, "http://localhost:8500/process-info/", req.URL.String())
				return &http.Response{
					StatusCode: sanitizeExternalSnapshotTest.edgeStackSidecarResponse,
					Body:       ioutil.NopCloser(bytes.NewBufferString(rawSidecarProcessInfoResponse)),
				}
			})

			snapshot, err := sanitizeExternalSnapshot(context.Background(), []byte(sanitizeExternalSnapshotTest.rawJSON), client)

			assert.Nil(innerT, err)
			assert.Equal(innerT, sanitizeExternalSnapshotTest.expectedSanitizedJSON, string(snapshot))
		})
	}
}

type roundTripFunc func(req *http.Request) *http.Response

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req), nil
}

func newHTTPTestClient(mockHandler roundTripFunc) *http.Client {
	return &http.Client{
		Transport: mockHandler,
	}
}
