package entrypoint_test

import (
	"fmt"
	"sort"
	"strings"
	"testing"

	"github.com/datawire/ambassador/cmd/ambex"
	"github.com/datawire/ambassador/cmd/entrypoint"
	envoy "github.com/datawire/ambassador/pkg/api/envoy/api/v2"
	bootstrap "github.com/datawire/ambassador/pkg/api/envoy/config/bootstrap/v2"
	"github.com/datawire/ambassador/pkg/snapshot/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The Fake struct is a test harness for edgestack. It spins up the key portions of the edgestack
// control plane that contain the bulk of its business logic, but instead of requiring tests to feed
// the business logic inputs via a real kubernetes or real consul deployment, inputs can be fed
// directly into the business logic via the harness APIs. This is not only several orders of
// magnitute faster, this also provides the author of the test perfect control over the ordering of
// events.
func TestFakeHello(t *testing.T) {
	// Use RunFake() to spin up the ambassador control plane with its inputs wired up to the Fake
	// APIs. This will automatically invoke the Setup() method for the Fake and also register the
	// Teardown() method with the Cleanup() hook of the supplied testing.T object.
	f := entrypoint.RunFake(t, entrypoint.FakeConfig{})

	// The Fake harness has a store for both kubernetes resources and consul endpoint data. We can
	// use the UpsertFile() to method to load as many resources as we would like. This is much like
	// doing a `kubectl apply` to a real kubernetes API server, however apply uses fancy merge
	// logic, whereas UpsertFile() does a simple Upsert operation. The `testdata/FakeHello.yaml`
	// file has a single mapping named "hello".
	f.UpsertFile("testdata/FakeHello.yaml")
	// Initially the Fake harness is paused. This means we can make as many method calls as we want
	// to in order to set up our initial conditions, and no inputs will be fed into the control
	// plane. To feed inputs to the control plane, we can choose to either manually invoke the
	// Flush() method whenever we want to send the control plane inputs, or for convenience we can
	// enable AutoFlush so that inputs are set whenever we modify data that the control plane is
	// watching.
	f.AutoFlush(true)

	// Once the control plane has started processing inputs, we need some way to observe its
	// computation. The Fake harness provides two ways to do this. The GetSnapshot() method allows
	// us to observe the snapshots assembled by the watchers for further processing. The
	// GetEnvoyConfig() method allows us to observe the envoy configuration produced from a
	// snapshot. Both these methods take a predicate so the can search for a snapshot that satisifes
	// whatever conditions are being tested. This allows the test to verify that the correct
	// computation is occurring without being overly prescriptive about the exact number of
	// snapshots and/or envoy configs that are produce to achieve a certain result.
	snap := f.GetSnapshot(func(snap *snapshot.Snapshot) bool {
		return len(snap.Kubernetes.Mappings) > 0
	})
	// Check that the snapshot contains the mapping from the file.
	assert.Equal(t, "hello", snap.Kubernetes.Mappings[0].Name)
}

// By default the Fake struct only invokes the first part of the pipeline that forms the control
// plane. If you use the EnvoyConfig option you can run the rest of the control plane. There is also
// a Timeout option that controls how long the harness waits for the desired Snapshot and/or
// EnvoyConfig to come along.
//
// Note that this test depends on diagd being in your path. If diagd is not available, the test will
// be skipped.
func TestFakeHelloWithEnvoyConfig(t *testing.T) {
	// Use the FakeConfig parameter to conigure the Fake harness. In this case we want to inspect
	// the EnvoyConfig that is produced from the inputs we feed the control plane.
	f := entrypoint.RunFake(t, entrypoint.FakeConfig{EnvoyConfig: true})

	// We will use the same inputs we used in TestFakeHello. A single mapping named "hello".
	f.UpsertFile("testdata/FakeHello.yaml")
	// Instead of using AutoFlush(true) we will manually Flush() when we want to feed inputs to the
	// control plane.
	f.Flush()

	// Grab the next snapshot that has mappings. The bootstrap logic should actually gaurantee this
	// is also the first mapping, but we aren't trying to test that here.
	snap := f.GetSnapshot(func(snap *snapshot.Snapshot) bool {
		return len(snap.Kubernetes.Mappings) > 0
	})
	// The first snapshot should contain the one and only mapping we have supplied the control
	// plane.x
	assert.Equal(t, "hello", snap.Kubernetes.Mappings[0].Name)

	// Create a predicate that will recognize the cluster we care about. The surjection from
	// Mappings to clusters is a bit opaque, so we just look for a cluster that contains the name
	// hello.
	isHelloCluster := func(c *envoy.Cluster) bool {
		return strings.Contains(c.Name, "hello")
	}

	// Grab the next envoy config that satisfies our predicate.
	envoyConfig := f.GetEnvoyConfig(func(envoy *bootstrap.Bootstrap) bool {
		return FindCluster(envoy, isHelloCluster) != nil
	})

	// Now let's dig into the envoy configuration and check that the correct target endpoint is
	// present.
	//
	// Note: This is admittedly quite verbose as envoy configuration is very dense. I expect we will
	// introduce an API that will provide a more abstract and convenient way of navigating envoy
	// configuration, however that will be covered in a future PR. The core of that logic is already
	// developing inside ambex since ambex slices and dices the envoy config in order to implement
	// RDS and enpdoint routing.
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

// This test will cover how to exercise the consul portion of the control plane. In principal it is
// the same as supplying kubernetes resources, however it uses the ConsulEndpoint() method to
// provide consul data.
func TestFakeHelloConsul(t *testing.T) {
	// Create our Fake harness and tell it to produce envoy configuration.
	f := entrypoint.RunFake(t, entrypoint.FakeConfig{EnvoyConfig: true})

	// Feed the control plane the kubernetes resources supplied in the referenced file. In this case
	// that includes a consul resolver and a mapping that uses that consul resolver.
	f.UpsertFile("testdata/FakeHelloConsul.yaml")
	// This test is a bit more interesting for the control plane from a bootstrapping perspective,
	// so we invoke Flush() manually rather than using AutoFlush(true). The control plane needs to
	// figure out that there is a mapping that depends on consul endpoint data, and it needs to wait
	// until that data is available before producing the first snapshot.
	f.Flush()

	// In prior tests we have only examined the snapshots that were ready to be processed, but the
	// watcher doesn't process every snapshot it constructs, it can discard various snapshots for
	// different reasons. Using GetSnapshotEntry() we can pull entries from the full log of
	// snapshots considered as opposed to just the skipping straight to the ones that are ready to
	// be processed.
	//
	// In this case the snapshot is considered incomplete until we supply enough consul endpoint
	// data for edgestack to construct an envoy config that won't send requests to our hello mapping
	// into a black hole.
	entry := f.GetSnapshotEntry(func(entry entrypoint.SnapshotEntry) bool {
		return entry.Disposition == entrypoint.SnapshotIncomplete && len(entry.Snapshot.Kubernetes.Mappings) > 0
	})
	// Check that the snapshot contains the mapping from the file.
	assert.Equal(t, "hello", entry.Snapshot.Kubernetes.Mappings[0].Name)

	// Now let's supply the endpoint data for the hello service referenced by our hello mapping.
	f.ConsulEndpoint("dc1", "hello", "1.2.3.4", 8080)
	f.Flush()
	// The Fake harness also tracks endpoints that get sent to ambex. We can use the GetEndpoints()
	// method to access them and check to see that the endpoint we supplied got delivered to ambex.
	endpoints := f.GetEndpoints(func(endpoints *ambex.Endpoints) bool {
		_, ok := endpoints.Entries["consul/dc1/hello"]
		return ok
	})
	assert.Len(t, endpoints.Entries, 1)
	assert.Equal(t, "1.2.3.4", endpoints.Entries["consul/dc1/hello"][0].Ip)

	// Grab the next snapshot that has mappings.
	snap := f.GetSnapshot(func(snap *snapshot.Snapshot) bool {
		return len(snap.Kubernetes.Mappings) > 0
	})

	// Check that the snapshot contains the mapping from the file.
	assert.Equal(t, "hello", snap.Kubernetes.Mappings[0].Name)

	// Check that our deltas are what we expect.
	assert.Equal(t, 2, len(snap.Deltas))

	deltaNames := []string{}

	for _, delta := range snap.Deltas {
		deltaNames = append(deltaNames, fmt.Sprintf("%s %s", delta.Kind, delta.Name))
	}

	sort.Strings(deltaNames)

	assert.Equal(t, []string{"ConsulResolver consul-dc1", "Mapping hello"}, deltaNames)

	// Create a predicate that will recognize the cluster we care about. The surjection from
	// Mappings to clusters is a bit opaque, so we just look for a cluster that contains the name
	// hello.
	isHelloCluster := func(c *envoy.Cluster) bool {
		return strings.Contains(c.Name, "hello")
	}

	// Grab the next envoy config that satisfies our predicate.
	envoyConfig := f.GetEnvoyConfig(func(envoy *bootstrap.Bootstrap) bool {
		return FindCluster(envoy, isHelloCluster) != nil
	})

	// Now let's check that the cluster produced properly references the endpoints that have already
	// arrived at ambex.
	cluster := FindCluster(envoyConfig, isHelloCluster)
	// It uses the consul resolver, so it should not embed the load assignment directly.
	assert.Nil(t, cluster.LoadAssignment)
	// It *should* have an EdsConfig.
	edsConfig := cluster.GetEdsClusterConfig()
	require.NotNil(t, edsConfig)
	// The EdsConfig *should* reference an endpoint.
	eps := endpoints.Entries[edsConfig.ServiceName]
	require.Len(t, eps, 1)
	// The endpoint it references *should* have our supplied ip address.
	assert.Equal(t, "1.2.3.4", eps[0].Ip)
}
