package entrypoint_test

import (
	"strings"
	"testing"

	"github.com/datawire/ambassador/cmd/entrypoint"
	envoy "github.com/datawire/ambassador/pkg/api/envoy/api/v2"
	bootstrap "github.com/datawire/ambassador/pkg/api/envoy/config/bootstrap/v2"
	"github.com/datawire/ambassador/pkg/snapshot/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFakeHello(t *testing.T) {
	// Spins up a fake ambassador that is running our real control plane.
	f := entrypoint.RunFake(t, entrypoint.FakeConfig{})

	// Feeds the control plane the kubernetes resources supplied in the referenced file.
	f.UpsertFile("testdata/FakeHello.yaml")
	f.AutoFlush(true)

	// Grab the next snapshot that satisfies the supplied predicate.
	snap := f.GetSnapshot(func(snap *snapshot.Snapshot) bool {
		return len(snap.Kubernetes.Mappings) > 0
	})
	// Check that the snapshot contains the mapping from the file.
	assert.Equal(t, "hello", snap.Kubernetes.Mappings[0].Name)
}

func TestFakeHelloWithEnvoyConfig(t *testing.T) {
	// Spins up a fake ambassador that is running our real control plane.
	f := entrypoint.RunFake(t, entrypoint.FakeConfig{EnvoyConfig: true})

	// Feeds the control plane the kubernetes resources supplied in the referenced file.
	f.UpsertFile("testdata/FakeHello.yaml")
	f.Flush()

	// Grab the next snapshot that satisfies the supplied predicate.
	snap := f.GetSnapshot(func(snap *snapshot.Snapshot) bool {
		return len(snap.Kubernetes.Mappings) > 0
	})
	// Check that the snapshot contains the mapping from the file.
	assert.Equal(t, "hello", snap.Kubernetes.Mappings[0].Name)

	// Create a predicate that will recognize the cluster we care about.
	isHelloCluster := func(c *envoy.Cluster) bool {
		return strings.Contains(c.Name, "hello")
	}

	// Grab the next envoy config that satisfies the supplied predicate.
	envoyConfig := f.GetEnvoyConfig(func(envoy *bootstrap.Bootstrap) bool {
		return FindCluster(envoy, isHelloCluster) != nil
	})

	cluster := FindCluster(envoyConfig, isHelloCluster)
	endpoints := cluster.LoadAssignment.Endpoints
	require.NotEmpty(t, endpoints)
	lbEndpoints := endpoints[0].LbEndpoints
	require.NotEmpty(t, lbEndpoints)
	endpoint := lbEndpoints[0].GetEndpoint()
	address := endpoint.Address.GetSocketAddress().Address
	assert.Equal(t, "hello", address)
}

func FindCluster(envoyConfig *bootstrap.Bootstrap, predicate func(*envoy.Cluster) bool) *envoy.Cluster {
	for _, cluster := range envoyConfig.StaticResources.Clusters {
		if predicate(cluster) {
			return cluster
		}
	}

	return nil
}

func TestFakeHelloConsul(t *testing.T) {
	// Spins up a fake ambassador that is running our real control plane.
	f := entrypoint.RunFake(t, entrypoint.FakeConfig{EnvoyConfig: true})

	// Feeds the control plane the kubernetes resources supplied in the referenced file. In this
	// case that includes a consul resolver.
	f.UpsertFile("testdata/FakeHelloConsul.yaml")
	f.Flush()
	f.ConsulEndpoint("dc1", "hello", "1.2.3.4", 8080)
	//f.ConsulEndpoint("dc1", "hello", "4.3.2.1", 8080)
	f.Flush()

	// Grab the next snapshot that satisfies the supplied predicate.
	snap := f.GetSnapshot(func(snap *snapshot.Snapshot) bool {
		return len(snap.Kubernetes.Mappings) > 0
	})
	// Check that the snapshot contains the mapping from the file.
	assert.Equal(t, "hello", snap.Kubernetes.Mappings[0].Name)

	// Create a predicate that will recognize the cluster we care about.
	isHelloCluster := func(c *envoy.Cluster) bool {
		return strings.Contains(c.Name, "hello")
	}

	// Grab the next envoy config that satisfies the supplied predicate.
	envoyConfig := f.GetEnvoyConfig(func(envoy *bootstrap.Bootstrap) bool {
		return FindCluster(envoy, isHelloCluster) != nil
	})

	cluster := FindCluster(envoyConfig, isHelloCluster)
	endpoints := cluster.LoadAssignment.Endpoints
	require.NotEmpty(t, endpoints)
	lbEndpoints := endpoints[0].LbEndpoints
	require.NotEmpty(t, lbEndpoints)

	endpoint := lbEndpoints[0].GetEndpoint()
	address := endpoint.Address.GetSocketAddress().Address
	assert.Equal(t, "1.2.3.4", address)

	/*	endpoint = lbEndpoints[1].GetEndpoint()
		address = endpoint.Address.GetSocketAddress().Address
		assert.Equal(t, "4.3.2.1", address)
	*/
}
