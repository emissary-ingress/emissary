package entrypoint_test

import (
	"fmt"
	"sort"
	"strings"
	"testing"

	"github.com/datawire/ambassador/cmd/entrypoint"
	envoy "github.com/datawire/ambassador/pkg/api/envoy/api/v2"
	bootstrap "github.com/datawire/ambassador/pkg/api/envoy/config/bootstrap/v2"
	"github.com/datawire/ambassador/pkg/snapshot/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestEndpointFiltering tests to be sure that endpoint changes are correctly filtered out
// when endpoint routing is not actually in use.
func TestEndpointFiltering(t *testing.T) {
	f := entrypoint.RunFake(t, entrypoint.FakeConfig{EnvoyConfig: true, DiagdDebug: false})
	f.AutoFlush(true)

	// XXX
	// Fake doesn't seem to really do namespacing or ambassadorID.

	f.UpsertYAML(`---
apiVersion: getambassador.io/v2
kind: Mapping
metadata:
  name:  qotm-mapping
spec:
  prefix: /qotm/
  service: qotm
`)
	f.UpsertFile("testdata/qotm-endpoints.yaml")

	// ...and we need a cluster for our mapping as well. At this point it should not be
	// using endpoint routing.
	assertQoTM(t, f, false)

	// Switch the QoTM mapping to explicitly use the endpoint resolver.
	f.UpsertYAML(`---
apiVersion: getambassador.io/v2
kind: Mapping
metadata:
  name:  qotm-mapping
spec:
  prefix: /qotm/
  service: qotm
  resolver: kubernetes-endpoint
`)

	assertQoTM(t, f, true)

	// Switch the QoTM mapping to use the service resolver.
	f.UpsertYAML(`---
apiVersion: getambassador.io/v2
kind: Mapping
metadata:
  name:  qotm-mapping
spec:
  prefix: /qotm/
  service: qotm
  resolver: kubernetes-service
`)

	assertQoTM(t, f, false)
}

func assertQoTM(t *testing.T, f *entrypoint.Fake, usingEndpoints bool) {
	// We need a snapshot with our mapping...
	snap := f.GetSnapshot(func(snap *snapshot.Snapshot) bool {
		return len(snap.Kubernetes.Mappings) > 0
	})

	fmt.Printf("====\nusingEndpoints %v:\n%s\n", usingEndpoints, Jsonify(snap))

	assert.Equal(t, "qotm-mapping", snap.Kubernetes.Mappings[0].Name)

	// Create a predicate that will recognize the cluster we care about. The surjection from
	// Mappings to clusters is a bit opaque, so we just look for a cluster that contains the name
	// "qotm".
	isQoTMCluster := func(c *envoy.Cluster) bool {
		return strings.Contains(c.Name, "qotm")
	}

	// Grab the next envoy config that satisfies our predicate.
	envoyConfig := f.GetEnvoyConfig(func(envoy *bootstrap.Bootstrap) bool {
		return FindCluster(envoy, isQoTMCluster) != nil
	})

	cluster := FindCluster(envoyConfig, isQoTMCluster)
	fmt.Printf("Cluster:\n%s\n", Jsonify(cluster))
	endpoints := cluster.LoadAssignment.Endpoints
	require.NotEmpty(t, endpoints)
	lbEndpoints := endpoints[0].LbEndpoints
	require.NotEmpty(t, lbEndpoints)

	addresses := []string{}

	for _, endpoint := range lbEndpoints {
		address := endpoint.GetEndpoint().Address.GetSocketAddress().Address
		addresses = append(addresses, address)
		sort.Strings(addresses)
	}

	if usingEndpoints {
		assert.Equal(t, len(addresses), 2)
		assert.Equal(t, addresses, []string{"10.42.0.15", "10.42.0.16"})
	} else {
		assert.Equal(t, len(addresses), 1)
		assert.Equal(t, addresses, []string{"qotm"})
	}
}
