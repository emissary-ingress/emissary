package entrypoint_test

import (
	"fmt"
	"sort"
	"strings"
	"testing"

	"github.com/datawire/ambassador/cmd/entrypoint"
	envoy "github.com/datawire/ambassador/pkg/api/envoy/api/v2"
	bootstrap "github.com/datawire/ambassador/pkg/api/envoy/config/bootstrap/v2"
	"github.com/datawire/ambassador/pkg/kates"
	"github.com/datawire/ambassador/pkg/snapshot/v1"
	"github.com/stretchr/testify/assert"
)

// TestEndpointFiltering tests to be sure that endpoint changes are correctly filtered out
// when endpoint routing is not actually in use.
func TestEndpointFiltering(t *testing.T) {
	f := entrypoint.RunFake(t, entrypoint.FakeConfig{EnvoyConfig: true, DiagdDebug: false})
	f.AutoFlush(true)

	// XXX
	// Fake doesn't seem to really do namespacing or ambassadorID.

	// ================
	STEP("START")

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

	// Here at the start of the test, we expect our mapping, no Endpoints, and no endpoint
	// deltas.
	assertEndpointsAndDeltas(t, f, &eadConfig{
		mappingName:         "qotm-mapping",
		clusterNameContains: "qotm",
		clusterAssignments:  []string{"qotm:80"},
	})

	// ================
	STEP("EXPLICIT ENDPOINT ROUTING")

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

	// Once that's done, we need the Mapping, an Endpoints, and an ADD Endpoints delta.
	assertEndpointsAndDeltas(t, f, &eadConfig{
		mappingName:         "qotm-mapping",
		clusterNameContains: "qotm",
		clusterAssignments:  []string{"10.42.0.15:5000", "10.42.0.16:5000"},
		k8sEndpointNames:    []string{"qotm"},
		deltaNames:          []string{"qotm"},
		deltaKinds:          []kates.DeltaType{kates.ObjectAdd},
	})

	// ================
	STEP("EXPLICIT SERVICE ROUTING")

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

	// Once that's done, we need the Mapping, no Endpoints, and a DELETE Endpoints delta.
	assertEndpointsAndDeltas(t, f, &eadConfig{
		mappingName:         "qotm-mapping",
		clusterNameContains: "qotm",
		clusterAssignments:  []string{"qotm:80"},
		deltaNames:          []string{"qotm"},
		deltaKinds:          []kates.DeltaType{kates.ObjectDelete},
	})

	// ================
	STEP("INSTALL CUSTOM RESOLVER")

	// Nothing should change, because nothing is using it yet.
	f.UpsertYAML(`---
apiVersion: getambassador.io/v2
kind: KubernetesEndpointResolver
metadata:
  name: custom-resolver
spec: {}
`)

	assertEndpointsAndDeltas(t, f, &eadConfig{
		mappingName:         "qotm-mapping",
		clusterNameContains: "qotm",
		clusterAssignments:  []string{"qotm:80"},
	})

	// ================
	STEP("INSTALL CUSTOM ENDPOINTS")

	// Again, nothing should change, because nothing is using endpoint routing right now.
	// So we should see a dropped snapshot entry (that still contains our mapping).
	f.UpsertFile("testdata/custom-endpoints.yaml")
	assertDroppedSnapshotEntry(t, f, "qotm-mapping")

	// ================
	STEP("SWITCH QOTM TO CUSTOM RESOLVER")

	// Once we switch the QotM Mapping to the custom resolver, we should see its Endpoints
	// plus the custom Endpoints we added last time.
	f.UpsertYAML(`---
apiVersion: getambassador.io/v2
kind: Mapping
metadata:
  name:  qotm-mapping
spec:
  prefix: /qotm/
  service: qotm
  resolver: custom-resolver
`)

	assertEndpointsAndDeltas(t, f, &eadConfig{
		mappingName:         "qotm-mapping",
		clusterNameContains: "qotm",
		clusterAssignments:  []string{"10.42.0.15:5000", "10.42.0.16:5000"},
		k8sEndpointNames:    []string{"qotm", "random-1", "random-2"},
		deltaNames:          []string{"qotm", "random-1", "random-2"},
		deltaKinds:          []kates.DeltaType{kates.ObjectAdd, kates.ObjectAdd, kates.ObjectAdd},
	})

	// ================
	STEP("DELETE random-1 ENDPOINTS")

	// When we delete the random-1 Endpoints, we should see a deletion delta for it, and
	// we should see that its Endpoints is gone.
	f.Delete("Endpoints", "default", "random-1")

	assertEndpointsAndDeltas(t, f, &eadConfig{
		mappingName:         "qotm-mapping",
		clusterNameContains: "qotm",
		clusterAssignments:  []string{"10.42.0.15:5000", "10.42.0.16:5000"},
		k8sEndpointNames:    []string{"qotm", "random-2"},
		deltaNames:          []string{"random-1"},
		deltaKinds:          []kates.DeltaType{kates.ObjectDelete},
	})

	// ================
	STEP("SWITCH QOTM TO DEFAULT RESOLVER")

	// Once we switch the QotM Mapping to the default resolver, we should see all the
	// Endpoints vanish, and we should see deletions for them.
	f.UpsertYAML(`---
apiVersion: getambassador.io/v2
kind: Mapping
metadata:
  name:  qotm-mapping
spec:
  prefix: /qotm/
  service: qotm
`)

	assertEndpointsAndDeltas(t, f, &eadConfig{
		mappingName:         "qotm-mapping",
		clusterNameContains: "qotm",
		clusterAssignments:  []string{"qotm:80"},
		deltaNames:          []string{"qotm", "random-2"},
		deltaKinds:          []kates.DeltaType{kates.ObjectDelete, kates.ObjectDelete},
	})

	// ================
	STEP("SWITCH DEFAULT RESOLVER TO CUSTOM RESOLVER")

	// Switching the default resolver to our custom resolver should make all the Endpoints
	// reappear, and we should see adds for them.
	f.UpsertYAML(`---
apiVersion: getambassador.io/v2
kind: Module
metadata:
  name:  ambassador
spec:
  config:
    resolver: custom-resolver
`)

	assertEndpointsAndDeltas(t, f, &eadConfig{
		mappingName:         "qotm-mapping",
		clusterNameContains: "qotm",
		clusterAssignments:  []string{"10.42.0.15:5000", "10.42.0.16:5000"},
		k8sEndpointNames:    []string{"qotm", "random-2"},
		deltaNames:          []string{"qotm", "random-2"},
		deltaKinds:          []kates.DeltaType{kates.ObjectAdd, kates.ObjectAdd},
	})

	// ================
	STEP("RE-ADD random-1 ENDPOINTS")

	// When we add the random-1 Endpoints again, by virtue of upserting the file with
	// both Endpoints in it, we should see an addition delta for it, and we should see
	// its Endpoints reappear.
	//
	// XXX Right now, we actually get _two_ deltas: an add for random-1 _and_ an update
	// for random-2. This happens because the K8s store doesn't check whether or not the
	// reapplied random-2 is different or not, it just calls it an update. At some point,
	// we might fix that, in which case this test will break.
	f.UpsertFile("testdata/custom-endpoints.yaml")

	assertEndpointsAndDeltas(t, f, &eadConfig{
		mappingName:         "qotm-mapping",
		clusterNameContains: "qotm",
		clusterAssignments:  []string{"10.42.0.15:5000", "10.42.0.16:5000"},
		k8sEndpointNames:    []string{"qotm", "random-1", "random-2"},
		deltaNames:          []string{"random-1", "random-2"},
		deltaKinds:          []kates.DeltaType{kates.ObjectAdd, kates.ObjectUpdate},
	})

	// ================
	STEP("SWITCH DEFAULT RESOLVER TO SERVICE RESOLVER")

	// Switching the default resolver back the service resolver should make all the Endpoints
	// Endpoints vanish, and we should see deletions for them.
	f.UpsertYAML(`---
apiVersion: getambassador.io/v2
kind: Module
metadata:
  name:  ambassador
spec:
  config:
    resolver: kubernetes-service
`)

	assertEndpointsAndDeltas(t, f, &eadConfig{
		mappingName:         "qotm-mapping",
		clusterNameContains: "qotm",
		clusterAssignments:  []string{"qotm:80"},
		deltaNames:          []string{"qotm", "random-1", "random-2"},
		deltaKinds:          []kates.DeltaType{kates.ObjectDelete, kates.ObjectDelete, kates.ObjectDelete},
	})

	// ================
	STEP("SWITCH DEFAULT RESOLVER TO ENDPOINT RESOLVER")

	// XXX This step will go away when we can assert that we didn't generate a snapshot.
	// But for now we'll see all three Endpoints reappear, with adds.
	f.UpsertYAML(`---
apiVersion: getambassador.io/v2
kind: Module
metadata:
  name:  ambassador
spec:
  config:
    resolver: kubernetes-endpoint
`)

	assertEndpointsAndDeltas(t, f, &eadConfig{
		mappingName:         "qotm-mapping",
		clusterNameContains: "qotm",
		clusterAssignments:  []string{"10.42.0.15:5000", "10.42.0.16:5000"},
		k8sEndpointNames:    []string{"qotm", "random-1", "random-2"},
		deltaNames:          []string{"qotm", "random-1", "random-2"},
		deltaKinds:          []kates.DeltaType{kates.ObjectAdd, kates.ObjectAdd, kates.ObjectAdd},
	})

	// ================
	STEP("DELETE AMBASSADOR MODULE")

	// XXX This step will change when we can assert that we didn't generate a snapshot.
	// For now, when we delete the Ambassador module, it'll implicitly flip the default
	// resolver back to the service resolver, so we'll see all the Endpoints vanish, and
	// we'll see deletes.
	f.Delete("Module", "default", "ambassador")

	assertEndpointsAndDeltas(t, f, &eadConfig{
		mappingName:         "qotm-mapping",
		clusterNameContains: "qotm",
		clusterAssignments:  []string{"qotm:80"},
		deltaNames:          []string{"qotm", "random-1", "random-2"},
		deltaKinds:          []kates.DeltaType{kates.ObjectDelete, kates.ObjectDelete, kates.ObjectDelete},
	})
}

func STEP(step string) {
	fmt.Printf("======== %s\n", step)
}

// eadConfig talks about endpoints and deltas.
type eadConfig struct {
	mappingName         string
	clusterNameContains string
	clusterAssignments  []string
	k8sEndpointNames    []string
	deltaNames          []string
	deltaKinds          []kates.DeltaType
}

// assertEndpointsAndDeltas asserts that:
// - we can get a snapshot and an Envoy config
// - the snapshot contains the Mapping named in eadConfig.mappingName
// - the snapshot's K8s Endpoints names match eadConfig.k8sEndpointNames
// - the snapshot contains Endpoints deltas with names matching eadConfig.deltaNames and
//   types matching eadConfig.deltaTypes.
// - the Envoy config contains a cluster whose name contains eadConfig.clusterNameContains
// - the cluster has load assignments that match eadConfig.clusterAssignments
func assertEndpointsAndDeltas(t *testing.T, f *entrypoint.Fake, ead *eadConfig) {
	// Make sure that we can get a snapshot, and that it contains our mapping...
	snap := assertSnapshotWithMapping(t, f, ead.mappingName)

	endpoints := snap.Kubernetes.Endpoints

	assert.Equal(t, len(ead.k8sEndpointNames), len(endpoints))

	epNames := []string{}

	for _, endpoint := range endpoints {
		epNames = append(epNames, endpoint.GetName())
	}

	sort.Strings(epNames)

	for i := range endpoints {
		assert.Equal(t, ead.k8sEndpointNames[i], epNames[i])
	}

	deltas := endpointDeltas(snap)

	assert.Equal(t, len(ead.deltaNames), len(deltas))
	assert.Equal(t, len(ead.deltaKinds), len(deltas))

	dNames := []string{}

	for _, delta := range deltas {
		dNames = append(dNames, delta.GetName())
	}

	sort.Strings(dNames)

	for i := range deltas {
		assert.Equal(t, ead.deltaNames[i], dNames[i])
		assert.Equal(t, ead.deltaKinds[i], deltas[i].DeltaType)
	}

	// Finally, make sure we have a properly-set-up cluster, too.
	assertEnvoyConfigWithCluster(t, f, ead)
}

// endpointDeltas grabs only the Endpoints deltas from the snapshot
func endpointDeltas(snap *snapshot.Snapshot) []*kates.Delta {
	deltas := []*kates.Delta{}

	for _, delta := range snap.Deltas {
		if delta.GroupVersionKind().Kind == "Endpoints" {
			deltas = append(deltas, delta)
		}
	}

	fmt.Printf("====== DELTAS:\n%s\n", Jsonify(deltas))

	return deltas
}

// assertSnapshotWithMapping grabs a snapshot and makes sure that it has a Mapping
// with the given name.
func assertSnapshotWithMapping(t *testing.T, f *entrypoint.Fake, mappingName string) *snapshot.Snapshot {
	// Assert that we can get a snapshot with a single Mapping with the given name.
	snap := f.GetSnapshot(func(snap *snapshot.Snapshot) bool {
		return len(snap.Kubernetes.Mappings) > 0
	})

	assert.NotNil(t, snap)

	fmt.Printf("==== SNAPSHOT:\n%s\n", Jsonify(snap))

	assert.Equal(t, mappingName, snap.Kubernetes.Mappings[0].Name)

	return snap
}

// assertEnvoyConfigWithCluster grabs an Envoy config and makes sure that it has a
// cluster containing a given name, with a given set of load assignments.
func assertEnvoyConfigWithCluster(t *testing.T, f *entrypoint.Fake, ead *eadConfig) {
	// Create a predicate that will recognize the cluster we care about. The surjection from
	// Mappings to clusters is a bit opaque, so we just look for a cluster that contains the name
	// "qotm".
	isWantedCluster := func(c *envoy.Cluster) bool {
		return strings.Contains(c.Name, ead.clusterNameContains)
	}

	// Grab the next envoy config that satisfies our predicate.
	envoyConfig := f.GetEnvoyConfig(func(envoy *bootstrap.Bootstrap) bool {
		return FindCluster(envoy, isWantedCluster) != nil
	})

	cluster := FindCluster(envoyConfig, isWantedCluster)
	fmt.Printf("Cluster:\n%s\n", Jsonify(cluster))
	endpoints := cluster.LoadAssignment.Endpoints

	assert.NotZero(t, len(endpoints))

	lbEndpoints := endpoints[0].LbEndpoints

	assignments := []string{}

	for _, endpoint := range lbEndpoints {
		address := endpoint.GetEndpoint().Address.GetSocketAddress().Address
		port := endpoint.GetEndpoint().Address.GetSocketAddress().GetPortValue()

		assignments = append(assignments, fmt.Sprintf("%s:%d", address, port))
	}

	sort.Strings(assignments)

	assert.Equal(t, len(ead.clusterAssignments), len(assignments))
	assert.Equal(t, ead.clusterAssignments, assignments)
}

func assertDroppedSnapshotEntry(t *testing.T, f *entrypoint.Fake, mappingName string) {
	entry := f.GetSnapshotEntry(func(entry entrypoint.SnapshotEntry) bool {
		fmt.Printf("Snapshot disposition %#v\n", entry.Disposition)
		return entry.Disposition == entrypoint.SnapshotDrop && len(entry.Snapshot.Kubernetes.Mappings) > 0
	})

	assert.Equal(t, mappingName, entry.Snapshot.Kubernetes.Mappings[0].Name)
}

func assertIncompleteSnapshotEntry(t *testing.T, f *entrypoint.Fake, mappingName string) {
	entry := f.GetSnapshotEntry(func(entry entrypoint.SnapshotEntry) bool {
		fmt.Printf("Snapshot disposition %#v\n", entry.Disposition)
		return entry.Disposition == entrypoint.SnapshotIncomplete && len(entry.Snapshot.Kubernetes.Mappings) > 0
	})

	assert.Equal(t, mappingName, entry.Snapshot.Kubernetes.Mappings[0].Name)
}
