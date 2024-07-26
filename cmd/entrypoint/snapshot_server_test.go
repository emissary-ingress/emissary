package entrypoint

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/datawire/dlib/dlog"
)

var sanitizeExternalSnapshotTests = []struct {
	testName              string
	rawJSON               string
	expectedSanitizedJSON string
}{
	{
		testName:              "no AmbassadorMeta",
		rawJSON:               `{"AmbassadorMeta":null,"Kubernetes":null,"Consul":null,"Deltas":null,"Invalid":null}`,
		expectedSanitizedJSON: `{"AmbassadorMeta":null,"Kubernetes":null,"Consul":null,"Deltas":null,"Invalid":null}`,
	},
	{
		testName:              "AmbassadorMeta with bad sidecar response",
		rawJSON:               `{"AmbassadorMeta":{"cluster_id":"","ambassador_id":"","ambassador_version":"","kube_version":""}}`,
		expectedSanitizedJSON: `{"AmbassadorMeta":{"cluster_id":"","ambassador_id":"","ambassador_version":"","kube_version":"","sidecar":null},"Kubernetes":null,"Consul":null,"Deltas":null,"Invalid":null}`,
	},
	{
		testName:              "AmbassadorMeta without any sidecar",
		rawJSON:               `{"AmbassadorMeta":{"cluster_id":"","ambassador_id":"","ambassador_version":"","kube_version":""}}`,
		expectedSanitizedJSON: `{"AmbassadorMeta":{"cluster_id":"","ambassador_id":"","ambassador_version":"","kube_version":"","sidecar":null},"Kubernetes":null,"Consul":null,"Deltas":null,"Invalid":null}`,
	},
}

func TestSanitizeExternalSnapshot(t *testing.T) {
	for _, sanitizeExternalSnapshotTest := range sanitizeExternalSnapshotTests {
		t.Run(sanitizeExternalSnapshotTest.testName, func(innerT *testing.T) {
			ctx := dlog.NewTestContext(t, false)
			snapshot, err := sanitizeExternalSnapshot(ctx, []byte(sanitizeExternalSnapshotTest.rawJSON))

			assert.Nil(innerT, err)
			assert.Equal(innerT, sanitizeExternalSnapshotTest.expectedSanitizedJSON, string(snapshot))
		})
	}
}
