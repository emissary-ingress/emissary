package entrypoint_test

import (
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
	"testing"

	"github.com/datawire/ambassador/v2/cmd/ambex"
	"github.com/datawire/ambassador/v2/cmd/entrypoint"
	v3bootstrap "github.com/datawire/ambassador/v2/pkg/api/envoy/config/bootstrap/v3"
	v3cluster "github.com/datawire/ambassador/v2/pkg/api/envoy/config/cluster/v3"
	"github.com/datawire/ambassador/v2/pkg/kates"
	"github.com/datawire/ambassador/v2/pkg/snapshot/v1"
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
	f := entrypoint.RunFake(t, entrypoint.FakeConfig{}, nil)

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
	f := entrypoint.RunFake(t, entrypoint.FakeConfig{EnvoyConfig: true}, nil)

	// We will use the same inputs we used in TestFakeHello. A single mapping named "hello".
	f.UpsertFile("testdata/FakeHello.yaml")
	// Instead of using AutoFlush(true) we will manually Flush() when we want to feed inputs to the
	// control plane.
	f.Flush()

	// Grab the next snapshot that has mappings. The v3bootstrap logic should actually gaurantee this
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
	isHelloCluster := func(c *v3cluster.Cluster) bool {
		return strings.Contains(c.Name, "hello")
	}

	// Grab the next envoy config that satisfies our predicate.
	envoyConfig := f.GetEnvoyConfig(func(envoy *v3bootstrap.Bootstrap) bool {
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

func FindCluster(envoyConfig *v3bootstrap.Bootstrap, predicate func(*v3cluster.Cluster) bool) *v3cluster.Cluster {
	for _, cluster := range envoyConfig.StaticResources.Clusters {
		if predicate(cluster) {
			return cluster
		}
	}

	return nil
}

func deltaSummary(snap *snapshot.Snapshot) []string {
	summary := []string{}

	var typestr string

	for _, delta := range snap.Deltas {
		switch delta.DeltaType {
		case kates.ObjectAdd:
			typestr = "add"
		case kates.ObjectUpdate:
			typestr = "update"
		case kates.ObjectDelete:
			typestr = "delete"
		default:
			panic("missing case")
		}

		summary = append(summary, fmt.Sprintf("%s %s %s", typestr, delta.Kind, delta.Name))
	}

	sort.Strings(summary)

	return summary
}

// This test will cover how to exercise the consul portion of the control plane. In principal it is
// the same as supplying kubernetes resources, however it uses the ConsulEndpoint() method to
// provide consul data.
func TestFakeHelloConsul(t *testing.T) {
	// This test will not pass in legacy mode because diagd will not emit EDS clusters in legacy mode.
	if legacy, err := strconv.ParseBool(os.Getenv("AMBASSADOR_LEGACY_MODE")); err == nil && legacy {
		return
	}

	os.Setenv("CONSULPORT", "8500")
	os.Setenv("CONSULHOST", "consul-1")

	// Create our Fake harness and tell it to produce envoy configuration.
	f := entrypoint.RunFake(t, entrypoint.FakeConfig{EnvoyConfig: true}, nil)

	// Feed the control plane the kubernetes resources supplied in the referenced file. In this case
	// that includes a consul resolver and a mapping that uses that consul resolver.
	f.UpsertFile("testdata/FakeHelloConsul.yaml")
	// This test is a bit more interesting for the control plane from a v3bootstrapping perspective,
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
	// ..and the TCPMapping as well
	assert.Equal(t, "hello-tcp", entry.Snapshot.Kubernetes.TCPMappings[0].Name)

	// Now let's supply the endpoint data for the hello service referenced by our hello mapping.
	f.ConsulEndpoint("dc1", "hello", "1.2.3.4", 8080)
	// And also supply the endpoint data for the hello-tcp service referenced by our hello mapping.
	f.ConsulEndpoint("dc1", "hello-tcp", "5.6.7.8", 3099)
	f.Flush()

	// The Fake harness also tracks endpoints that get sent to ambex. We can use the GetEndpoints()
	// method to access them and check to see that the endpoint we supplied got delivered to ambex.
	endpoints := f.GetEndpoints(func(endpoints *ambex.Endpoints) bool {
		_, ok := endpoints.Entries["consul/dc1/hello"]
		if ok {
			_, okTcp := endpoints.Entries["consul/dc1/hello-tcp"]
			return okTcp
		}
		return false
	})
	assert.Len(t, endpoints.Entries, 2)
	assert.Equal(t, "1.2.3.4", endpoints.Entries["consul/dc1/hello"][0].Ip)

	// Grab the next snapshot that has both mappings, tcpmappings, and a Consul resolver. The v3bootstrap logic
	// should actually guarantee this is also the first mapping, but we aren't trying to test
	// that here.
	snap := f.GetSnapshot(func(snap *snapshot.Snapshot) bool {
		return (len(snap.Kubernetes.Mappings) > 0) && (len(snap.Kubernetes.TCPMappings) > 0) && (len(snap.Kubernetes.ConsulResolvers) > 0)
	})
	// The first snapshot should contain both the mapping and tcpmapping we have supplied the control
	// plane.
	assert.Equal(t, "hello", snap.Kubernetes.Mappings[0].Name)
	assert.Equal(t, "hello-tcp", snap.Kubernetes.TCPMappings[0].Name)

	// It should also contain one ConsulResolver with a Spec.Address of
	// "consul-server.default:8500" (where the 8500 came from an environment variable).
	assert.Equal(t, "consul-server.default:8500", snap.Kubernetes.ConsulResolvers[0].Spec.Address)

	// Check that our deltas are what we expect.
	assert.Equal(t, []string{"add AmbassadorMapping hello", "add AmbassadorTCPMapping hello-tcp", "add ConsulResolver consul-dc1"}, deltaSummary(snap))

	// Create a predicate that will recognize the cluster we care about. The surjection from
	// Mappings to clusters is a bit opaque, so we just look for a cluster that contains the name
	// hello.
	isHelloTCPCluster := func(c *v3cluster.Cluster) bool {
		return strings.Contains(c.Name, "hello_tcp")
	}
	isHelloCluster := func(c *v3cluster.Cluster) bool {
		return strings.Contains(c.Name, "hello") && !isHelloTCPCluster(c)
	}

	// Grab the next envoy config that satisfies our predicate.
	envoyConfig := f.GetEnvoyConfig(func(envoy *v3bootstrap.Bootstrap) bool {
		return FindCluster(envoy, isHelloCluster) != nil
	})

	// Now let's check that the cluster produced properly references the endpoints that have already
	// arrived at ambex.
	cluster := FindCluster(envoyConfig, isHelloCluster)
	assert.NotNil(t, cluster)
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

	// Finally, let's check that the TCP cluster is OK too.
	cluster = FindCluster(envoyConfig, isHelloTCPCluster)
	assert.NotNil(t, cluster)
	// It uses the consul resolver, so it should not embed the load assignment directly.
	assert.Nil(t, cluster.LoadAssignment)
	// It *should* have an EdsConfig.
	edsConfig = cluster.GetEdsClusterConfig()
	require.NotNil(t, edsConfig)
	// The EdsConfig *should* reference an endpoint.
	eps = endpoints.Entries[edsConfig.ServiceName]
	require.Len(t, eps, 1)
	// The endpoint it references *should* have our supplied ip address.
	assert.Equal(t, "5.6.7.8", eps[0].Ip)

	// Next up, change the Consul resolver definition.
	f.UpsertYAML(`
---
apiVersion: getambassador.io/v2
kind: ConsulResolver
metadata:
  name: consul-dc1
spec:
  address: $CONSULHOST:$CONSULPORT
  datacenter: dc1
`)
	f.Flush()

	// Repeat the snapshot checks. We must have mappings and consulresolvers...
	snap = f.GetSnapshot(func(snap *snapshot.Snapshot) bool {
		return (len(snap.Kubernetes.Mappings) > 0) && (len(snap.Kubernetes.TCPMappings) > 0) && (len(snap.Kubernetes.ConsulResolvers) > 0)
	})

	// ...with one delta, namely the ConsulResolver...
	assert.Equal(t, []string{"update ConsulResolver consul-dc1"}, deltaSummary(snap))

	// ...where the mapping name hasn't changed...
	assert.Equal(t, "hello", snap.Kubernetes.Mappings[0].Name)
	assert.Equal(t, "hello-tcp", snap.Kubernetes.TCPMappings[0].Name)

	// ...but the Consul server address has.
	assert.Equal(t, "consul-1:8500", snap.Kubernetes.ConsulResolvers[0].Spec.Address)

	// Finally, delete the Consul resolver, then replace it. This is mostly just testing that
	// things don't crash.

	f.Delete("ConsulResolver", "default", "consul-dc1")
	f.Flush()

	f.UpsertYAML(`
---
apiVersion: getambassador.io/v2
kind: ConsulResolver
metadata:
  name: consul-dc1
spec:
  address: $CONSULHOST:9999
  datacenter: dc1
`)
	f.Flush()

	// Repeat all the checks.
	snap = f.GetSnapshot(func(snap *snapshot.Snapshot) bool {
		return (len(snap.Kubernetes.Mappings) > 0) && (len(snap.Kubernetes.TCPMappings) > 0) && (len(snap.Kubernetes.ConsulResolvers) > 0)
	})

	// Two deltas here since we've deleted and re-added without a check in between.
	// (They appear out of order here because of string sorting. Don't panic.)
	assert.Equal(t, []string{"add ConsulResolver consul-dc1", "delete ConsulResolver consul-dc1"}, deltaSummary(snap))

	// ...one mapping...
	assert.Equal(t, "hello", snap.Kubernetes.Mappings[0].Name)
	assert.Equal(t, "hello-tcp", snap.Kubernetes.TCPMappings[0].Name)

	// ...and one ConsulResolver.
	assert.Equal(t, "consul-1:9999", snap.Kubernetes.ConsulResolvers[0].Spec.Address)
}
