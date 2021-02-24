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

func TestConsulEndpointFiltering(t *testing.T) {
	f := entrypoint.RunFake(t, entrypoint.FakeConfig{EnvoyConfig: true, DiagdDebug: true})

	// ================
	STEP("INITIALIZE K8s")

	// Start with a Mapping and the consul-dc1 resolver. This shouldn't yet produce
	// a snapshot, since the system should decide that Consul isn't yet bootstrapped.
	f.UpsertFile("testdata/FakeHelloConsul.yaml")
	f.Flush()

	assertIncompleteSnapshotEntry(t, f, "hello")

	// ================
	STEP("INITIALIZE Consul")

	f.ConsulEndpoint("dc1", "hello", "1.2.3.4", 8080)
	f.Flush()

	// At this point we should see a configuration that's using endpoint routing, but we
	// should have no K8s Deltas.
	assertEndpointsAndDeltas(t, f, &eadConfig{
		mappingName:         "hello",
		clusterNameContains: "hello",
		clusterAssignments:  []string{"1.2.3.4:8080"},
		consulEndpointNames: []string{"hello"},
		consulAddresses:     []string{"dc1/hello/1.2.3.4:8080"},
	})

	// ================
	STEP("ADD ENDPOINTS to K8s")

	// When we add some Endpoints to K8s, we should see nothing, since we're not using
	// the K8s endpoint resolver. We will, however, see a new snapshot, since we're adding
	// a Service to K8s too.

	f.UpsertFile("testdata/hello-endpoints.yaml")
	f.Flush()

	assertEndpointsAndDeltas(t, f, &eadConfig{
		mappingName:         "hello",
		clusterNameContains: "hello",
		clusterAssignments:  []string{"1.2.3.4:8080"},
		consulEndpointNames: []string{"hello"},
		consulAddresses:     []string{"dc1/hello/1.2.3.4:8080"},
	})

	// ================
	STEP("ADD ENDPOINT to Consul")

	// XXX This currently does an add. It would be nice to have update, delete, etc.
	f.ConsulEndpoint("dc1", "hello", "4.3.2.1", 8080)
	f.Flush()

	// At this point we should see a configuration that's using endpoint routing, but we
	// should have no K8s Deltas.
	assertEndpointsAndDeltas(t, f, &eadConfig{
		mappingName:         "hello",
		clusterNameContains: "hello",
		clusterAssignments:  []string{"1.2.3.4:8080", "4.3.2.1:8080"},
		consulEndpointNames: []string{"hello"},
		consulAddresses:     []string{"dc1/hello/1.2.3.4:8080", "dc1/hello/4.3.2.1:8080"},
	})

	// ================
	STEP("SWITCH TO K8s ENDPOINT ROUTING")

	// Switch the hello mapping to explicitly use the endpoint resolver.
	f.UpsertYAML(`---
apiVersion: getambassador.io/v2
kind: Mapping
metadata:
  name: hello
  namespace: default
spec:
  prefix: /hello
  service: hello
  resolver: kubernetes-endpoint
`)
	f.Flush()

	// At this point we should see the K8s endpoints appear, with deltas, and we should see
	// the load assignments for our cluster switch. We'll still see the Consul endpoints,
	// though, since the Consul resolver is present.
	//
	// XXX Is that really correct? Feels like the Consul endpoints should disappear here.
	assertEndpointsAndDeltas(t, f, &eadConfig{
		mappingName:         "hello",
		clusterNameContains: "hello",
		clusterAssignments:  []string{"10.42.0.15:5000", "10.42.0.16:5000"},
		k8sEndpointNames:    []string{"hello"},
		deltaNames:          []string{"hello"},
		deltaKinds:          []kates.DeltaType{kates.ObjectAdd},
		consulEndpointNames: []string{"hello"},
		consulAddresses:     []string{"dc1/hello/1.2.3.4:8080", "dc1/hello/4.3.2.1:8080"},
	})

	// ================
	STEP("DROP CONSUL RESOLVER")

	// Delete the ConsulResolver.
	f.Delete("ConsulResolver", "default", "consul-dc1")
	f.Flush()

	// At this point we should see no K8s changes, but the Consul endpoints should disappear.
	//
	// XXX At the moment, the Consul resolver leaves its endpoints in place, even though it
	// shouldn't.
	assertEndpointsAndDeltas(t, f, &eadConfig{
		mappingName:         "hello",
		clusterNameContains: "hello",
		clusterAssignments:  []string{"10.42.0.15:5000", "10.42.0.16:5000"},
		k8sEndpointNames:    []string{"hello"},
		consulEndpointNames: []string{"hello"},
		consulAddresses:     []string{"dc1/hello/1.2.3.4:8080", "dc1/hello/4.3.2.1:8080"},
	})

	// ================
	STEP("SWITCH TO K8s DEFAULT ROUTING")

	// Switch the hello mapping to use the service resolver, by default.
	f.UpsertYAML(`---
apiVersion: getambassador.io/v2
kind: Mapping
metadata:
  name: hello
  namespace: default
spec:
  prefix: /hello
  service: hello
`)
	f.Flush()

	// At this point we should see the K8s endpoints disappear, with deltas, and we should
	// see the load assignments for our cluster switch.
	//
	// XXX At the moment we'll still see the Consul endpoints.
	assertEndpointsAndDeltas(t, f, &eadConfig{
		mappingName:         "hello",
		clusterNameContains: "hello",
		clusterAssignments:  []string{"hello:80"},
		deltaNames:          []string{"hello"},
		deltaKinds:          []kates.DeltaType{kates.ObjectDelete},
		consulEndpointNames: []string{"hello"},
		consulAddresses:     []string{"dc1/hello/1.2.3.4:8080", "dc1/hello/4.3.2.1:8080"},
	})

	// ================
	STEP("SWITCH DEFAULT RESOLVER TO CONSUL RESOLVER")

	// Put the Consul resolver back, and switch the default resolver to it. We should see
	// the load assignments for the cluster switch back to the Consul endpoints, which
	// should be present in the snapshot. There should be no K8s Endpoints and thus no
	// deltas, since we just switched back to the service resolver.
	//
	// XXX In pratice, you can't really use the Consul resolver as a default resolver:
	// Edge Stack adds mappings that have "." in their service names. For this test,
	// though, that's fine.

	f.UpsertFile("testdata/FakeHelloConsul.yaml")
	f.UpsertYAML(`---
apiVersion: getambassador.io/v2
kind: Module
metadata:
  name:  ambassador
spec:
  config:
    resolver: consul-dc1
`)
	f.Flush() // get all the changes applied at once

	assertEndpointsAndDeltas(t, f, &eadConfig{
		mappingName:         "hello",
		clusterNameContains: "hello",
		clusterAssignments:  []string{"1.2.3.4:8080", "4.3.2.1:8080"},
		consulEndpointNames: []string{"hello"},
		consulAddresses:     []string{"dc1/hello/1.2.3.4:8080", "dc1/hello/4.3.2.1:8080"},
	})

	// ================
	STEP("SWITCH BACK TO MAPPING CONSUL RESOLVER")

	// Switch the default resolver back to the K8s service resolver, and switch our
	// mapping back to the Consul dc1 resolver. This is just to get us back into a
	// state that we would fully expect Ambassador to work correctly. We should see
	// no changes, because we explicitly flush both changes at once.

	f.Delete("Module", "default", "ambassador")
	f.UpsertYAML(`---
apiVersion: getambassador.io/v2
kind: Mapping
metadata:
  name: hello
  namespace: default
spec:
  prefix: /hello
  service: hello
  resolver: consul-dc1
`)
	f.Flush() // get all the changes applied at once

	assertEndpointsAndDeltas(t, f, &eadConfig{
		mappingName:         "hello",
		clusterNameContains: "hello",
		clusterAssignments:  []string{"1.2.3.4:8080", "4.3.2.1:8080"},
		consulEndpointNames: []string{"hello"},
		consulAddresses:     []string{"dc1/hello/1.2.3.4:8080", "dc1/hello/4.3.2.1:8080"},
	})

	// ================
	STEP("ADD CONSUL other-dc RESOLVER")

	// Add a second Consul resolver, with DC other-dc, and give it some endpoints.
	// We'll get a snapshot generated here, but it'll look identical to the last one.
	// XXX _Should_ we get a snapshot generated here? Shouldn't we ignore this because
	// no Mappings use it?

	f.UpsertYAML(`---
apiVersion: getambassador.io/v2
kind: ConsulResolver
metadata:
  name: consul-otherdc
spec:
  address: consul-server.default.svc.cluster.local:8500
  datacenter: other-dc
`)
	f.ConsulEndpoint("other-dc", "hello", "1.2.1.2", 8000)
	f.Flush() // get all the changes applied at once

	assertEndpointsAndDeltas(t, f, &eadConfig{
		mappingName:         "hello",
		clusterNameContains: "hello",
		clusterAssignments:  []string{"1.2.3.4:8080", "4.3.2.1:8080"},
		consulEndpointNames: []string{"hello"},
		consulAddresses:     []string{"dc1/hello/1.2.3.4:8080", "dc1/hello/4.3.2.1:8080"},
	})

	// XXXXXXXX We can't actually do this next test right now: it simply doesn't
	// work to trigger a Kube change with Consul knock-on effects. In practical terms,
	// this is a bug in the watcher loop that will affect Ambassador users trying to
	// switch from one DC to another.

	// 	// ================
	// 	STEP("SWITCH TO CONSUL other-dc RESOLVER")

	// 	// Switch our Mapping to our new other-dc Consul resolver. We should see the load
	// 	// assignments for the cluster switch to the Consul endpoints from other-dc, which
	// 	// should be present in the snapshot. There should be no K8s Endpoints and thus no
	// 	// deltas.
	// 	//
	// 	// XXX This doesn't actually work right now unless we trigger a _Consul_ change in
	// 	// addition to the Kube change we actually need. That's a bug which must be fixed
	// 	// later.
	// 	//
	// 	// XXX We should also see the Consul endpoints from dc1 vanish.

	// 	f.UpsertYAML(`---
	// apiVersion: getambassador.io/v2
	// kind: Mapping
	// metadata:
	//   name: hello
	//   namespace: default
	// spec:
	//   prefix: /hello
	//   service: hello
	//   resolver: consul-otherdc
	// `)
	// 	// f.Flush()

	// 	// assertEndpointsAndDeltas(t, f, &eadConfig{
	// 	// 	mappingName:         "hello",
	// 	// 	clusterNameContains: "hello",
	// 	// 	clusterAssignments:  []string{"1.2.3.4:8080", "4.3.2.1:8080"},
	// 	// 	consulEndpointNames: []string{"hello"},
	// 	// 	consulAddresses:     []string{"dc1/hello/1.2.3.4:8080", "dc1/hello/4.3.2.1:8080"},
	// 	// })

	// 	f.ConsulEndpoint("other-dc", "hello", "2.1.2.1", 8080)
	// 	f.Flush()

	// 	assertEndpointsAndDeltas(t, f, &eadConfig{
	// 		mappingName:         "hello",
	// 		clusterNameContains: "hello",
	// 		clusterAssignments:  []string{"1.2.1.2:8000", "2.1.2.1:8080"},
	// 		consulEndpointNames: []string{"hello"},
	// 		consulAddresses:     []string{"dc1/hello/1.2.3.4:8080", "dc1/hello/4.3.2.1:8080"},
	// 	})
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
	consulEndpointNames []string
	consulAddresses     []string
	deltaNames          []string
	deltaKinds          []kates.DeltaType
}

// deltaNameAndType is a simple struct to manage delta names and types (it makes for
// less typing for the test writer).
type deltaNameAndType struct {
	name      string // for sorting
	deltaType kates.DeltaType
}

// assertEndpointsAndDeltas asserts that:
// - we can get a snapshot and an Envoy config
// - the snapshot contains the Mapping named in eadConfig.mappingName
// - the snapshot's K8s Endpoints names match eadConfig.k8sEndpointNames
// - the snapshot's Consul endpoint names match eadConfig.consulEndpointNames
// - the snapshot's Consul endpoint addresses and ports match eadConfig.consulAddresses
// - the snapshot contains Endpoints deltas with names matching eadConfig.deltaNames and
//   types matching eadConfig.deltaTypes.
// - the Envoy config contains a cluster whose name contains eadConfig.clusterNameContains
// - the cluster has load assignments that match eadConfig.clusterAssignments
func assertEndpointsAndDeltas(t *testing.T, f *entrypoint.Fake, ead *eadConfig) {
	// Make sure that we can get a snapshot, and that it contains our mapping...
	snap := assertSnapshotWithMapping(t, f, ead.mappingName)

	// Given the snapshot, grab the K8s Endpoints...
	k8sEndpoints := snap.Kubernetes.Endpoints

	// ...make sure all the lengths match up...
	assert.Equal(t, len(ead.k8sEndpointNames), len(k8sEndpoints))

	// ...then build an array of the actual endpoint names, both to format things
	// politely and to sort it

	actualNames := []string{}

	for _, endpoint := range k8sEndpoints {
		actualNames = append(actualNames, endpoint.GetName())
	}

	sort.Strings(actualNames)

	// If we have any expected names...
	if ead.k8sEndpointNames != nil {
		// ...then make a copy so we can sort it...
		expectedNames := make([]string, len(ead.k8sEndpointNames))
		copy(expectedNames, ead.k8sEndpointNames)
		sort.Strings(expectedNames)

		// ...and make sure the names match up.
		assert.Equal(t, expectedNames, actualNames)
	} else {
		// No expected names, so we need to have no actual names.
		assert.Zero(t, len(actualNames))
	}

	// Consul endpoints are smarter than Kube Endpoints. They're a dict mapping
	// service names to a list of endpoints, each of which has a service name, an
	// address, and a port. We format that as "dc/service/address:port" for our
	// work here, then sort the whole list.

	consulEndpoints := snap.Consul.Endpoints

	assert.Equal(t, len(ead.consulEndpointNames), len(consulEndpoints))

	epAddresses := []string{}

	for _, consulService := range consulEndpoints {
		for _, endpoint := range consulService.Endpoints {
			fqaddr := fmt.Sprintf("%s/%s/%s:%d", consulService.Id, endpoint.Service, endpoint.Address, endpoint.Port)
			epAddresses = append(epAddresses, fqaddr)
		}
	}

	sort.Strings(epAddresses)

	// OK, if we have expected Consul endpoints...
	if ead.consulAddresses != nil {
		// ...then once again we need to copy and sort.
		expectedAddresses := make([]string, len(ead.consulAddresses))
		copy(expectedAddresses, ead.consulAddresses)
		sort.Strings(expectedAddresses)

		assert.Equal(t, expectedAddresses, epAddresses)
	} else {
		// And, again, if we expect no addresses, we need to see no addresses.
		assert.Zero(t, len(epAddresses))
	}

	// Next up, do it all over again for deltas. Start by grabbing the
	// (sorted) list of relevant deltas from the snapshot.

	deltas := endpointDeltas(snap)

	// Next, we make sure all the lengths match up...
	assert.Equal(t, len(ead.deltaNames), len(deltas))
	assert.Equal(t, len(ead.deltaKinds), len(deltas))

	// ...then build up an array of deltaNamesAndTypes to sort.
	dnat := make([]*deltaNameAndType, len(deltas))

	for i := range deltas {
		dnat[i] = &deltaNameAndType{
			name:      ead.deltaNames[i],
			deltaType: ead.deltaKinds[i],
		}
	}

	sort.SliceStable(dnat, func(i, j int) bool {
		return dnat[i].name < dnat[j].name
	})

	// After all that, we can check to make sure everything matches.
	for i := range deltas {
		assert.Equal(t, dnat[i].name, deltas[i].GetName())
		assert.Equal(t, dnat[i].deltaType, deltas[i].DeltaType)
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

	// We want a sorted return here.
	sort.SliceStable(deltas, func(i, j int) bool {
		return deltas[i].GetName() < deltas[j].GetName()
	})

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

	// Find the cluster we want...
	cluster := FindCluster(envoyConfig, isWantedCluster)
	fmt.Printf("Cluster:\n%s\n", Jsonify(cluster))

	// ...and pull out its load assignments, which is weirder than it should be.
	endpoints := cluster.LoadAssignment.Endpoints

	assert.NotZero(t, len(endpoints))

	lbEndpoints := endpoints[0].LbEndpoints

	// Format everything neatly, both for readability and for sorting.
	assignments := []string{}

	for _, endpoint := range lbEndpoints {
		address := endpoint.GetEndpoint().Address.GetSocketAddress().Address
		port := endpoint.GetEndpoint().Address.GetSocketAddress().GetPortValue()

		assignments = append(assignments, fmt.Sprintf("%s:%d", address, port))
	}

	sort.Strings(assignments)

	// Make sure the lengths match...
	assert.Equal(t, len(ead.clusterAssignments), len(assignments))

	// ...then make a shallow copy of the expected assignments so that we can sort
	// that too.
	expectedAssignments := make([]string, len(assignments))
	copy(expectedAssignments, ead.clusterAssignments)
	sort.Strings(expectedAssignments)

	// Finally, make sure the values match.
	assert.Equal(t, expectedAssignments, assignments)
}

// assertDroppedSnapshotEntry asserts that we've dropped a snapshot entry that contained
// the named Mapping.
func assertDroppedSnapshotEntry(t *testing.T, f *entrypoint.Fake, mappingName string) {
	entry := f.GetSnapshotEntry(func(entry entrypoint.SnapshotEntry) bool {
		fmt.Printf("Snapshot disposition %#v\n", entry.Disposition)
		return entry.Disposition == entrypoint.SnapshotDrop && len(entry.Snapshot.Kubernetes.Mappings) > 0
	})

	assert.Equal(t, mappingName, entry.Snapshot.Kubernetes.Mappings[0].Name)
}

// assertIncompleteSnapshotEntry asserts that we had an incomplete snapshot entry that
// contained the named Mapping.
func assertIncompleteSnapshotEntry(t *testing.T, f *entrypoint.Fake, mappingName string) {
	entry := f.GetSnapshotEntry(func(entry entrypoint.SnapshotEntry) bool {
		fmt.Printf("Snapshot disposition %#v\n", entry.Disposition)
		return entry.Disposition == entrypoint.SnapshotIncomplete && len(entry.Snapshot.Kubernetes.Mappings) > 0
	})

	assert.Equal(t, mappingName, entry.Snapshot.Kubernetes.Mappings[0].Name)
}
