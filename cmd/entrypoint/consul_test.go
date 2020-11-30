package entrypoint

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	amb "github.com/datawire/ambassador/pkg/api/getambassador.io/v2"
	"github.com/datawire/ambassador/pkg/consulwatch"
	"github.com/datawire/ambassador/pkg/kates"
)

const manifests = `
---
apiVersion: getambassador.io/v2
kind: ConsulResolver
metadata:
  name: consultest-resolver
spec:
  ambassador_id: consultest
  address: consultest-consul:8500
  datacenter: dc1
---
apiVersion: ambassador/v1
kind:  Mapping
name:  consultest_k8s_mapping
prefix: /consultest_k8s/
service: consultest-http-k8s
---
apiVersion: getambassador.io/v1
kind: KubernetesServiceResolver
name: kubernetes-service
---
apiVersion: getambassador.io/v1
kind: KubernetesEndpointResolver
name: endpoint
---
apiVersion: ambassador/v1
kind:  Mapping
name:  consultest_consul_mapping
prefix: /consultest_consul/
service: consultest-consul-service
# tls: consultest-client-context # this doesn't seem to work... ambassador complains with "no private key in secret ..."
resolver: consultest-resolver
load_balancer:
  policy: round_robin
---
apiVersion: ambassador/v1
kind:  TLSContext
name:  consultest-client-context
secret: consultest-client-cert-secret
`

func TestReconcile(t *testing.T) {
	resolvers, mappings, c, tw := setup(t)
	c.reconcile(resolvers, mappings)
	tw.Assert("consultest-resolver.default:consultest-consul-service:watch")
	extra := &amb.Mapping{
		Spec: amb.MappingSpec{
			Service:  "foo",
			Resolver: "consultest-resolver",
		},
	}
	extra.SetNamespace("default")
	c.reconcile(resolvers, append(mappings, extra))
	tw.Assert(
		"consultest-resolver.default:foo:watch",
	)
	c.reconcile(resolvers, nil)
	tw.Assert(
		"consultest-resolver.default:consultest-consul-service:stop",
		"consultest-resolver.default:foo:stop",
	)
}

func TestCleanup(t *testing.T) {
	resolvers, mappings, c, tw := setup(t)
	c.reconcile(resolvers, mappings)
	tw.Assert("consultest-resolver.default:consultest-consul-service:watch")
	c.cleanup()
	tw.Assert("consultest-resolver.default:consultest-consul-service:stop")
}

func TestBootstrap(t *testing.T) {
	resolvers, mappings, c, _ := setup(t)
	assert.False(t, c.isBootstrapped())
	c.reconcile(resolvers, mappings)
	assert.False(t, c.isBootstrapped())
	// XXX: break this (maybe use a chan to replace uncoalesced dirties and passing con around?)
	c.endpoints["consultest-consul-service"] = consulwatch.Endpoints{}
	assert.True(t, c.isBootstrapped())
}

func setup(t *testing.T) (resolvers []*amb.ConsulResolver, mappings []*amb.Mapping, c *consul, tw *testWatcher) {
	objs, err := kates.ParseManifests(manifests)
	require.NoError(t, err)

	parent := &kates.Unstructured{}
	parent.SetNamespace("default")

	for _, obj := range objs {
		obj = convertAnnotation(parent, obj)
		obj.SetNamespace("default")
		switch o := obj.(type) {
		case *amb.ConsulResolver:
			resolvers = append(resolvers, o)
		case *amb.Mapping:
			mappings = append(mappings, o)
		}
	}

	assert.Equal(t, 1, len(resolvers))
	assert.Equal(t, 2, len(mappings))

	tw = &testWatcher{t: t, events: make(map[string]bool)}
	c = newConsul(context.TODO(), tw)
	tw.Assert()

	return
}

type testWatcher struct {
	t      *testing.T
	events map[string]bool
}

func (tw *testWatcher) Log(event string) {
	tw.events[event] = true
}

func (tw *testWatcher) Logf(format string, args ...interface{}) {
	tw.Log(fmt.Sprintf(format, args...))
}

func (tw *testWatcher) Assert(events ...string) {
	eventsMap := make(map[string]bool)
	for _, e := range events {
		eventsMap[e] = true
	}
	assert.Equal(tw.t, eventsMap, tw.events)
	tw.events = make(map[string]bool)
}

func (tw *testWatcher) Watch(resolver *amb.ConsulResolver, mapping *amb.Mapping, _ chan consulwatch.Endpoints) Stopper {
	rname := fmt.Sprintf("%s.%s", resolver.GetName(), resolver.GetNamespace())
	svc := mapping.Spec.Service
	tw.Logf("%s:%s:watch", rname, svc)
	return &testStopper{watcher: tw, resolver: rname, service: svc}
}

type testStopper struct {
	watcher  *testWatcher
	resolver string
	service  string
}

func (ts *testStopper) Stop() {
	ts.watcher.Logf("%s:%s:stop", ts.resolver, ts.service)
}
