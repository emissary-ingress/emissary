package entrypoint_test

import (
	"fmt"
	"net"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/util/intstr"

	"github.com/datawire/ambassador/v2/cmd/ambex"
	"github.com/datawire/ambassador/v2/cmd/entrypoint"
	v3bootstrap "github.com/datawire/ambassador/v2/pkg/api/envoy/config/bootstrap/v3"
	v3cluster "github.com/datawire/ambassador/v2/pkg/api/envoy/config/cluster/v3"
	amb "github.com/datawire/ambassador/v2/pkg/api/getambassador.io/v3alpha1"
	"github.com/datawire/ambassador/v2/pkg/kates"
	snapshotTypes "github.com/datawire/ambassador/v2/pkg/snapshot/v1"
)

func TestEndpointRouting(t *testing.T) {
	f := entrypoint.RunFake(t, entrypoint.FakeConfig{EnvoyConfig: true}, nil)
	// Create Mapping, Service, and Endpoints resources to start.
	assert.NoError(t, f.Upsert(makeMapping("default", "foo", "/foo", "foo", "endpoint")))
	assert.NoError(t, f.Upsert(makeService("default", "foo")))
	subset, err := makeSubset(8080, "1.2.3.4")
	require.NoError(t, err)
	assert.NoError(t, f.Upsert(makeEndpoints("default", "foo", subset)))
	f.Flush()
	snap, err := f.GetSnapshot(HasMapping("default", "foo"))
	require.NoError(t, err)
	assert.NotNil(t, snap)

	// Check that the endpoints resource we created at the start was properly propagated.
	endpoints, err := f.GetEndpoints(HasEndpoints("k8s/default/foo"))
	require.NoError(t, err)
	assert.Equal(t, "1.2.3.4", endpoints.Entries["k8s/default/foo"][0].Ip)
	assert.Equal(t, uint32(8080), endpoints.Entries["k8s/default/foo"][0].Port)
	assert.Contains(t, endpoints.Entries, "k8s/default/foo/80")
	assert.Equal(t, "1.2.3.4", endpoints.Entries["k8s/default/foo/80"][0].Ip)
	assert.Equal(t, uint32(8080), endpoints.Entries["k8s/default/foo/80"][0].Port)
}

func TestEndpointRoutingMappingAnnotations(t *testing.T) {
	f := entrypoint.RunFake(t, entrypoint.FakeConfig{EnvoyConfig: true}, nil)
	// Create Mapping, Service, and Endpoints resources to start.
	svc := makeService("default", "foo")
	svc.ObjectMeta.Annotations = map[string]string{
		"getambassador.io/config": `
---
apiVersion: getambassador.io/v3alpha1
kind: Mapping
name: foo
prefix: /foo
service: foo
resolver: endpoint`,
	}
	assert.NoError(t, f.Upsert(svc))
	subset, err := makeSubset(8080, "1.2.3.4")
	require.NoError(t, err)
	assert.NoError(t, f.Upsert(makeEndpoints("default", "foo", subset)))
	f.Flush()
	snap, err := f.GetSnapshot(HasService("default", "foo"))
	require.NoError(t, err)
	assert.NotNil(t, snap)

	// Check that the endpoints resource we created at the start was properly propagated.
	endpoints, err := f.GetEndpoints(HasEndpoints("k8s/default/foo"))
	require.NoError(t, err)
	assert.Equal(t, "1.2.3.4", endpoints.Entries["k8s/default/foo"][0].Ip)
	assert.Equal(t, uint32(8080), endpoints.Entries["k8s/default/foo"][0].Port)
	assert.Contains(t, endpoints.Entries, "k8s/default/foo/80")
	assert.Equal(t, "1.2.3.4", endpoints.Entries["k8s/default/foo/80"][0].Ip)
	assert.Equal(t, uint32(8080), endpoints.Entries["k8s/default/foo/80"][0].Port)
}

func TestEndpointRoutingMultiplePorts(t *testing.T) {
	f := entrypoint.RunFake(t, entrypoint.FakeConfig{EnvoyConfig: true}, nil)
	// Create Mapping, Service, and Endpoints, except this time the Service has multiple ports.
	assert.NoError(t, f.Upsert(makeMapping("default", "foo", "/foo", "foo", "endpoint")))
	assert.NoError(t, f.Upsert(&kates.Service{
		TypeMeta:   kates.TypeMeta{Kind: "Service"},
		ObjectMeta: kates.ObjectMeta{Namespace: "default", Name: "foo"},
		Spec: kates.ServiceSpec{
			Ports: []kates.ServicePort{
				{
					Name:       "cleartext",
					Port:       80,
					TargetPort: intstr.FromInt(8080),
				},
				{
					Name:       "encrypted",
					Port:       443,
					TargetPort: intstr.FromInt(8443),
				},
			},
		},
	}))
	subset, err := makeSubset("cleartext", 8080, "encrypted", 8443, "1.2.3.4")
	require.NoError(t, err)
	assert.NoError(t, f.Upsert(makeEndpoints("default", "foo", subset)))
	f.Flush()
	snap, err := f.GetSnapshot(HasMapping("default", "foo"))
	require.NoError(t, err)
	assert.NotNil(t, snap)

	// Check that the endpoints resource we created at the start was properly propagated.
	endpoints, err := f.GetEndpoints(HasEndpoints("k8s/default/foo/80"))
	require.NoError(t, err)
	assert.Contains(t, endpoints.Entries, "k8s/default/foo/80")
	assert.Contains(t, endpoints.Entries, "k8s/default/foo/443")
	assert.Contains(t, endpoints.Entries, "k8s/default/foo/cleartext")
	assert.Contains(t, endpoints.Entries, "k8s/default/foo/encrypted")

	// Make sure 80 and cleartext both map to container port 8080
	assert.Equal(t, "1.2.3.4", endpoints.Entries["k8s/default/foo/80"][0].Ip)
	assert.Equal(t, uint32(8080), endpoints.Entries["k8s/default/foo/80"][0].Port)

	assert.Equal(t, "1.2.3.4", endpoints.Entries["k8s/default/foo/cleartext"][0].Ip)
	assert.Equal(t, uint32(8080), endpoints.Entries["k8s/default/foo/cleartext"][0].Port)

	// Make sure 443 and encrypted both map to container port 8443
	assert.Equal(t, "1.2.3.4", endpoints.Entries["k8s/default/foo/443"][0].Ip)
	assert.Equal(t, uint32(8443), endpoints.Entries["k8s/default/foo/443"][0].Port)

	assert.Equal(t, "1.2.3.4", endpoints.Entries["k8s/default/foo/encrypted"][0].Ip)
	assert.Equal(t, uint32(8443), endpoints.Entries["k8s/default/foo/encrypted"][0].Port)
}

func TestEndpointRoutingIP(t *testing.T) {
	f := entrypoint.RunFake(t, entrypoint.FakeConfig{EnvoyConfig: true}, nil)
	// Create a Mapping that points straight at an IP address.
	assert.NoError(t, f.Upsert(makeMapping("default", "foo", "/foo", "4.3.2.1", "endpoint")))
	f.Flush()

	// Check that the envoy config embeds the IP address directly in the cluster config.
	config, err := f.GetEnvoyConfig(func(config *v3bootstrap.Bootstrap) bool {
		return FindCluster(config, ClusterNameContains("4_3_2_1")) != nil
	})
	require.NoError(t, err)
	cluster := FindCluster(config, ClusterNameContains("4_3_2_1"))
	require.NotNil(t, cluster)
	require.NotNil(t, cluster.LoadAssignment)
	require.Len(t, cluster.LoadAssignment.Endpoints, 1)
	require.Len(t, cluster.LoadAssignment.Endpoints[0].LbEndpoints, 1)
	ep := cluster.LoadAssignment.Endpoints[0].LbEndpoints[0].GetEndpoint()
	assert.NotNil(t, ep)
	sockAddr := ep.Address.GetSocketAddress()
	assert.Equal(t, "4.3.2.1", sockAddr.Address)
}

// Test that we resend endpoints when a new mapping is created that references an existing set of
// endpoints.
func TestEndpointRoutingMappingCreation(t *testing.T) {
	f := entrypoint.RunFake(t, entrypoint.FakeConfig{}, nil)
	assert.NoError(t, f.Upsert(makeService("default", "foo")))
	subset, err := makeSubset(8080, "1.2.3.4")
	require.NoError(t, err)
	assert.NoError(t, f.Upsert(makeEndpoints("default", "foo", subset)))
	f.Flush()
	f.AssertEndpointsEmpty(timeout)
	assert.NoError(t, f.UpsertYAML(`
---
apiVersion: getambassador.io/v3alpha1
kind: Mapping
metadata:
  name: foo
  namespace: default
spec:
  prefix: /foo
  resolver: endpoint
  service: foo.default
`))
	f.Flush()
	// Check that endpoints get sent even though we did not do actually update endpoints between
	// this flush and the previous one.
	endpoints, err := f.GetEndpoints(HasEndpoints("k8s/default/foo/80"))
	require.NoError(t, err)
	assert.Equal(t, "1.2.3.4", endpoints.Entries["k8s/default/foo/80"][0].Ip)
}

func ClusterNameContains(substring string) func(*v3cluster.Cluster) bool {
	return func(c *v3cluster.Cluster) bool {
		return strings.Contains(c.Name, substring)
	}
}

func HasService(namespace, name string) func(snapshot *snapshotTypes.Snapshot) bool {
	return func(snapshot *snapshotTypes.Snapshot) bool {
		for _, m := range snapshot.Kubernetes.Services {
			if m.Namespace == namespace && m.Name == name {
				return true
			}
		}
		return false
	}
}

func HasMapping(namespace, name string) func(snapshot *snapshotTypes.Snapshot) bool {
	return func(snapshot *snapshotTypes.Snapshot) bool {
		for _, m := range snapshot.Kubernetes.Mappings {
			if m.Namespace == namespace && m.Name == name {
				return true
			}
		}
		return false
	}
}

func HasEndpoints(path string) func(endpoints *ambex.Endpoints) bool {
	return func(endpoints *ambex.Endpoints) bool {
		_, ok := endpoints.Entries[path]
		return ok
	}
}

func makeMapping(namespace, name, prefix, service, resolver string) *amb.Mapping {
	return &amb.Mapping{
		TypeMeta:   kates.TypeMeta{Kind: "Mapping"},
		ObjectMeta: kates.ObjectMeta{Namespace: namespace, Name: name},
		Spec: amb.MappingSpec{
			Prefix:   prefix,
			Service:  service,
			Resolver: resolver,
		},
	}
}

func makeService(namespace, name string) *kates.Service {
	return &kates.Service{
		TypeMeta:   kates.TypeMeta{Kind: "Service"},
		ObjectMeta: kates.ObjectMeta{Namespace: namespace, Name: name},
		Spec: kates.ServiceSpec{
			Ports: []kates.ServicePort{
				{
					Port:       80,
					TargetPort: intstr.FromInt(8080),
				},
			},
		},
	}
}

func makeEndpoints(namespace, name string, subsets ...kates.EndpointSubset) *kates.Endpoints {
	return &kates.Endpoints{
		TypeMeta:   kates.TypeMeta{Kind: "Endpoints"},
		ObjectMeta: kates.ObjectMeta{Namespace: namespace, Name: name},
		Subsets:    subsets,
	}
}

// makeSubset provides a convenient way to kubernetes EndpointSubset resources. Any int args are
// ports, any ip address strings are addresses, and no ip address strings are used as the port name
// for any ports that follow them in the arg list.
func makeSubset(args ...interface{}) (kates.EndpointSubset, error) {
	portName := ""
	var ports []kates.EndpointPort
	var addrs []kates.EndpointAddress
	for _, arg := range args {
		switch v := arg.(type) {
		case int:
			ports = append(ports, kates.EndpointPort{Name: portName, Port: int32(v), Protocol: kates.ProtocolTCP})
		case string:
			IP := net.ParseIP(v)
			if IP == nil {
				portName = v
			} else {
				addrs = append(addrs, kates.EndpointAddress{IP: v})
			}
		default:
			return kates.EndpointSubset{}, fmt.Errorf("unrecognized type: %T", v)
		}
	}

	return kates.EndpointSubset{Addresses: addrs, Ports: ports}, nil
}
