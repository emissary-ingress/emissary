package entrypoint_test

import (
	"fmt"
	"net"
	"reflect"
	"strings"
	"testing"

	"github.com/datawire/ambassador/cmd/ambex"
	"github.com/datawire/ambassador/cmd/entrypoint"
	envoy "github.com/datawire/ambassador/pkg/api/envoy/api/v2"
	bootstrap "github.com/datawire/ambassador/pkg/api/envoy/config/bootstrap/v2"
	v2 "github.com/datawire/ambassador/pkg/api/getambassador.io/v2"
	"github.com/datawire/ambassador/pkg/kates"
	"github.com/datawire/ambassador/pkg/snapshot/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func TestEndpointRouting(t *testing.T) {
	f := entrypoint.RunFake(t, entrypoint.FakeConfig{EnvoyConfig: true}, nil)
	// Create Mapping, Service, and Endpoints resources to start.
	f.Upsert(makeMapping("default", "foo", "/foo", "foo", "endpoint"))
	f.Upsert(makeService("default", "foo"))
	f.Upsert(makeEndpoints("default", "foo", makeSubset(8080, "1.2.3.4")))
	f.Flush()
	snap := f.GetSnapshot(HasMapping("default", "foo"))
	assert.NotNil(t, snap)

	// Check that the endpoints resource we created at the start was properly propagated.
	endpoints := f.GetEndpoints(HasEndpoints("k8s/default/foo"))
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
apiVersion: getambassador.io/v2
kind: Mapping
name: foo
prefix: /foo
service: foo
resolver: endpoint`,
	}
	f.Upsert(svc)
	f.Upsert(makeEndpoints("default", "foo", makeSubset(8080, "1.2.3.4")))
	f.Flush()
	snap := f.GetSnapshot(HasService("default", "foo"))
	assert.NotNil(t, snap)

	// Check that the endpoints resource we created at the start was properly propagated.
	endpoints := f.GetEndpoints(HasEndpoints("k8s/default/foo"))
	assert.Equal(t, "1.2.3.4", endpoints.Entries["k8s/default/foo"][0].Ip)
	assert.Equal(t, uint32(8080), endpoints.Entries["k8s/default/foo"][0].Port)
	assert.Contains(t, endpoints.Entries, "k8s/default/foo/80")
	assert.Equal(t, "1.2.3.4", endpoints.Entries["k8s/default/foo/80"][0].Ip)
	assert.Equal(t, uint32(8080), endpoints.Entries["k8s/default/foo/80"][0].Port)
}

func TestEndpointRoutingMultiplePorts(t *testing.T) {
	f := entrypoint.RunFake(t, entrypoint.FakeConfig{EnvoyConfig: true}, nil)
	// Create Mapping, Service, and Endpoints, except this time the Service has multiple ports.
	f.Upsert(makeMapping("default", "foo", "/foo", "foo", "endpoint"))
	f.Upsert(&kates.Service{
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
	})
	f.Upsert(makeEndpoints("default", "foo", makeSubset("cleartext", 8080, "encrypted", 8443, "1.2.3.4")))
	f.Flush()
	snap := f.GetSnapshot(HasMapping("default", "foo"))
	assert.NotNil(t, snap)

	// Check that the endpoints resource we created at the start was properly propagated.
	endpoints := f.GetEndpoints(HasEndpoints("k8s/default/foo/80"))
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
	f.Upsert(makeMapping("default", "foo", "/foo", "4.3.2.1", "endpoint"))
	f.Flush()

	// Check that the envoy config embeds the IP address directly in the cluster config.
	config := f.GetEnvoyConfig(func(config *bootstrap.Bootstrap) bool {
		return FindCluster(config, ClusterNameContains("4_3_2_1")) != nil
	})
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
	f.Upsert(makeService("default", "foo"))
	f.Upsert(makeEndpoints("default", "foo", makeSubset(8080, "1.2.3.4")))
	f.Flush()
	f.AssertEndpointsEmpty(timeout)
	f.UpsertYAML(`
---
apiVersion: getambassador.io/v2
kind: Mapping
metadata:
  name: foo
  namespace: default
spec:
  prefix: /foo
  resolver: endpoint
  service: foo.default
`)
	f.Flush()
	// Check that endpoints get sent even though we did not do actually update endpoints between
	// this flush and the previous one.
	endpoints := f.GetEndpoints(HasEndpoints("k8s/default/foo/80"))
	assert.Equal(t, "1.2.3.4", endpoints.Entries["k8s/default/foo/80"][0].Ip)
}

func ClusterNameContains(substring string) func(*envoy.Cluster) bool {
	return func(c *envoy.Cluster) bool {
		return strings.Contains(c.Name, substring)
	}
}

func HasService(namespace, name string) func(snapshot *snapshot.Snapshot) bool {
	return func(snapshot *snapshot.Snapshot) bool {
		for _, m := range snapshot.Kubernetes.Services {
			if m.Namespace == namespace && m.Name == name {
				return true
			}
		}
		return false
	}
}

func HasMapping(namespace, name string) func(snapshot *snapshot.Snapshot) bool {
	return func(snapshot *snapshot.Snapshot) bool {
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

func makeMapping(namespace, name, prefix, service, resolver string) *v2.Mapping {
	return &v2.Mapping{
		TypeMeta:   kates.TypeMeta{Kind: "Mapping"},
		ObjectMeta: kates.ObjectMeta{Namespace: namespace, Name: name},
		Spec: v2.MappingSpec{
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
func makeSubset(args ...interface{}) kates.EndpointSubset {
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
			panic(fmt.Sprintf("unrecognized type: %v", reflect.TypeOf(v)))
		}
	}

	return kates.EndpointSubset{Addresses: addrs, Ports: ports}
}
