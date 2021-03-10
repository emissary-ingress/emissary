package entrypoint_test

import (
	"fmt"
	"sort"
	"strings"

	"github.com/datawire/ambassador/cmd/entrypoint"
	envoy "github.com/datawire/ambassador/pkg/api/envoy/api/v2"
	bootstrap "github.com/datawire/ambassador/pkg/api/envoy/config/bootstrap/v2"
	v2 "github.com/datawire/ambassador/pkg/api/getambassador.io/v2"
	"github.com/datawire/ambassador/pkg/kates"
	"github.com/datawire/ambassador/pkg/snapshot/v1"
	"github.com/stretchr/testify/assert"
)

// eadConfig talks about endpoints and deltas.
type eadConfig struct {
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

// snapshotGetMapping returns the Mapping with a given name inside a snapshot.
// If no mapping is found, it returns nil.
//
// It's used here as a building block for predicates passed to f.GetSnapshot and
// f.GetSnapshotEntry.
func snapshotGetMapping(snap *snapshot.Snapshot, mappingName string) *v2.Mapping {
	for _, mapping := range snap.Kubernetes.Mappings {
		if mapping.GetName() == mappingName {
			return mapping
		}
	}

	return nil
}

// getSnapshotContainingMapping grabs a snapshot and makes sure that it has a Mapping
// with the given name.
func getSnapshotContainingMapping(f *entrypoint.Fake, mappingName string) *snapshot.Snapshot {
	// Grab the first snapshot that contains a mapping with the given name.
	snap := f.GetSnapshot(func(snap *snapshot.Snapshot) bool {
		return snapshotGetMapping(snap, mappingName) != nil
	})

	assert.NotNil(f.T, snap)

	fmt.Printf("==== SNAPSHOT:\n%s\n", Jsonify(snap))

	assert.Equal(f.T, mappingName, snap.Kubernetes.Mappings[0].Name)

	return snap
}

// getEnvoyConfigAndCluster grabs the first Envoy config it can find that
// contains a particular cluster, and returns the Envoy config and the specific
// Envoy cluster.
//
// clusterNamePart can be any substring that uniquely identifies the cluster
// you want. You certainly could use the whole cluster name, but that's often
// annoying to figure out.
func getEnvoyConfigAndCluster(f *entrypoint.Fake, clusterNamePart string) (*bootstrap.Bootstrap, *envoy.Cluster) {
	// Create a predicate that will recognize the cluster we care about. The surjection from
	// Mappings to clusters is a bit opaque, so we just look for a cluster that contains the name
	// "qotm".
	isWantedCluster := func(c *envoy.Cluster) bool {
		return strings.Contains(c.Name, clusterNamePart)
	}

	// Grab the next envoy config that satisfies our predicate.
	envoyConfig := f.GetEnvoyConfig(func(envoy *bootstrap.Bootstrap) bool {
		return FindCluster(envoy, isWantedCluster) != nil
	})

	// Find the cluster we want...
	cluster := FindCluster(envoyConfig, isWantedCluster)
	fmt.Printf("Cluster:\n%s\n", Jsonify(cluster))

	return envoyConfig, cluster
}

// assertEndpointsAndDeltas asserts that:
// - the given snapshot's K8s Endpoints names match eadConfig.k8sEndpointNames
// - the snapshot's Consul endpoint names match eadConfig.consulEndpointNames
// - the snapshot's Consul endpoint addresses and ports match eadConfig.consulAddresses
// - the snapshot contains Endpoints deltas with names matching eadConfig.deltaNames and
//   types matching eadConfig.deltaTypes.
// - the given Envoy cluster has load assignments that match eadConfig.clusterAssignments
func assertEndpointsAndDeltas(f *entrypoint.Fake, snap *snapshot.Snapshot, cluster *envoy.Cluster, ead *eadConfig) {
	// Given the snapshot, grab the K8s Endpoints...
	k8sEndpoints := snap.Kubernetes.Endpoints

	// ...make sure all the lengths match up...
	assert.Equal(f.T, len(ead.k8sEndpointNames), len(k8sEndpoints))

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
		assert.Equal(f.T, expectedNames, actualNames)
	} else {
		// No expected names, so we need to have no actual names.
		assert.Zero(f.T, len(actualNames))
	}

	// Consul endpoints are smarter than Kube Endpoints. They're a dict mapping
	// service names to a list of endpoints, each of which has a service name, an
	// address, and a port. We format that as "dc/service/address:port" for our
	// work here, then sort the whole list.

	consulEndpoints := snap.Consul.Endpoints

	assert.Equal(f.T, len(ead.consulEndpointNames), len(consulEndpoints))

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

		assert.Equal(f.T, expectedAddresses, epAddresses)
	} else {
		// And, again, if we expect no addresses, we need to see no addresses.
		assert.Zero(f.T, len(epAddresses))
	}

	// Next up, do it all over again for deltas. Start by grabbing the
	// (sorted) list of relevant deltas from the snapshot.

	deltas := endpointDeltas(snap)

	// Next, we make sure all the lengths match up...
	assert.Equal(f.T, len(ead.deltaNames), len(deltas))
	assert.Equal(f.T, len(ead.deltaKinds), len(deltas))

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
		assert.Equal(f.T, dnat[i].name, deltas[i].GetName())
		assert.Equal(f.T, dnat[i].deltaType, deltas[i].DeltaType)
	}

	// Finally, make sure the cluster's load assignments match, too.
	assertClusterLoadAssignments(f, cluster, ead)
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

// assertClusterLoadAssignments asserts that the load assignments for a given cluster match
// what's specified in eadConfig.
func assertClusterLoadAssignments(f *entrypoint.Fake, cluster *envoy.Cluster, ead *eadConfig) {
	// Pull out the cluster's load assignments, which is weirder than it should be.
	endpoints := cluster.LoadAssignment.Endpoints

	assert.NotZero(f.T, len(endpoints))

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
	assert.Equal(f.T, len(ead.clusterAssignments), len(assignments))

	// ...then make a shallow copy of the expected assignments so that we can sort
	// that too.
	expectedAssignments := make([]string, len(assignments))
	copy(expectedAssignments, ead.clusterAssignments)
	sort.Strings(expectedAssignments)

	// Finally, make sure the values match.
	assert.Equal(f.T, expectedAssignments, assignments)
}

// getDroppedEntryContainingMapping gets the next dropped snapshot entry that
// contains the named Mapping.
func getDroppedEntryContainingMapping(f *entrypoint.Fake, mappingName string) {
	entry := f.GetSnapshotEntry(func(entry entrypoint.SnapshotEntry) bool {
		if entry.Disposition != entrypoint.SnapshotDrop {
			return false
		}

		return snapshotGetMapping(entry.Snapshot, mappingName) != nil
	})

	assert.Equal(f.T, mappingName, entry.Snapshot.Kubernetes.Mappings[0].Name)
}

// getIncompleteEntryContainingMapping gets the next incomplete snapshot entry that
// contains the named Mapping.
func getIncompleteEntryContainingMapping(f *entrypoint.Fake, mappingName string) *entrypoint.SnapshotEntry {
	entry := f.GetSnapshotEntry(func(entry entrypoint.SnapshotEntry) bool {
		if entry.Disposition != entrypoint.SnapshotIncomplete {
			return false
		}

		return snapshotGetMapping(entry.Snapshot, mappingName) != nil
	})

	assert.Equal(f.T, mappingName, entry.Snapshot.Kubernetes.Mappings[0].Name)

	return &entry
}
