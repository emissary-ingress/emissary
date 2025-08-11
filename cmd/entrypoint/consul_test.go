package entrypoint

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/datawire/dlib/dgroup"
	"github.com/datawire/dlib/dlog"
	amb "github.com/emissary-ingress/emissary/v3/pkg/api/emissary-ingress.dev/v4alpha1"
	"github.com/emissary-ingress/emissary/v3/pkg/consulwatch"
	"github.com/emissary-ingress/emissary/v3/pkg/kates"
	snapshotTypes "github.com/emissary-ingress/emissary/v3/pkg/snapshot/v1"
)

const manifests = `
---
apiVersion: getambassador.io/v3alpha1
kind: ConsulResolver
metadata:
  name: consultest-resolver
spec:
  ambassador_id:
   - consultest
  address: consultest-consul:8500
  datacenter: dc1
---
apiVersion: getambassador.io/v3alpha1
kind: Mapping
name:  consultest_k8s_mapping
prefix: /consultest_k8s/
service: consultest-http-k8s
---
apiVersion: getambassador.io/v3alpha1
kind: TCPMapping
name:  consultest_k8s_mapping_tcp
port: 3099
service: consultest-http-k8s
---
apiVersion: getambassador.io/v2
kind: KubernetesServiceResolver
name: kubernetes-service
---
apiVersion: getambassador.io/v2
kind: KubernetesEndpointResolver
name: endpoint
---
apiVersion: getambassador.io/v3alpha1
kind: Mapping
name:  consultest_consul_mapping
prefix: /consultest_consul/
service: consultest-consul-service
# tls: consultest-client-context # this doesn't seem to work... ambassador complains with "no private key in secret ..."
resolver: consultest-resolver
load_balancer:
  policy: round_robin
---
apiVersion: getambassador.io/v3alpha1
kind: TCPMapping
name:  consultest_consul_mapping_tcp
port: 3090
service: consultest-consul-service-tcp
resolver: consultest-resolver
---
apiVersion: getambassador.io/v3alpha1
kind:  TLSContext
name:  consultest-client-context
secret: consultest-client-cert-secret
`

func TestReconcile(t *testing.T) {
	ctx, resolvers, mappings, c, tw := setup(t)
	require.NoError(t, c.reconcile(ctx, resolvers, mappings))
	tw.Assert(
		"consultest-resolver.default:consultest-consul-service:watch",
		"consultest-resolver.default:consultest-consul-service-tcp:watch",
	)
	extra := consulMapping{
		Service:  "foo",
		Resolver: "consultest-resolver",
	}
	require.NoError(t, c.reconcile(ctx, resolvers, append(mappings, extra)))
	tw.Assert(
		"consultest-resolver.default:foo:watch",
	)
	require.NoError(t, c.reconcile(ctx, resolvers, nil))
	tw.Assert(
		"consultest-resolver.default:consultest-consul-service-tcp:stop",
		"consultest-resolver.default:consultest-consul-service:stop",
		"consultest-resolver.default:foo:stop",
	)
}

func TestCleanup(t *testing.T) {
	ctx, resolvers, mappings, c, tw := setup(t)
	require.NoError(t, c.reconcile(ctx, resolvers, mappings))
	tw.Assert(
		"consultest-resolver.default:consultest-consul-service:watch",
		"consultest-resolver.default:consultest-consul-service-tcp:watch",
	)
	require.NoError(t, c.cleanup(ctx))
	tw.Assert(
		"consultest-resolver.default:consultest-consul-service:stop",
		"consultest-resolver.default:consultest-consul-service-tcp:stop",
	)
}

func TestBootstrap(t *testing.T) {
	ctx, resolvers, mappings, c, _ := setup(t)
	assert.False(t, c.isBootstrapped())
	require.NoError(t, c.reconcile(ctx, resolvers, mappings))
	assert.False(t, c.isBootstrapped())
	// XXX: break this (maybe use a chan to replace uncoalesced dirties and passing con around?)
	//
	// In order for consul to be considered bootstrapped, both the service referenced by
	// a Mapping and the one refereced by a TCPMapping should have Endpoints{
	c.endpoints["consultest-consul-service"] = consulwatch.Endpoints{}
	c.endpoints["consultest-consul-service-tcp"] = consulwatch.Endpoints{}
	assert.True(t, c.isBootstrapped())
}

func setup(t *testing.T) (ctx context.Context, resolvers []*amb.ConsulResolver, mappings []consulMapping, c *consulWatcher, tw *testWatcher) {
	var cancel context.CancelFunc
	ctx, cancel = context.WithCancel(dlog.NewTestContext(t, false))
	grp := dgroup.NewGroup(ctx, dgroup.GroupConfig{})
	t.Cleanup(func() {
		cancel()
		assert.NoError(t, grp.Wait())
	})

	parent := &kates.Unstructured{
		Object: map[string]interface{}{
			"metadata": map[string]interface{}{
				"namespace": "default",
				"annotations": map[string]interface{}{
					"getambassador.io/config": manifests,
				},
			},
		},
	}

	objs, err := snapshotTypes.ParseAnnotationResources(parent)
	require.NoError(t, err)

	for _, obj := range objs {
		newobj, err := snapshotTypes.ValidateAndConvertObject(ctx, obj)
		if !assert.NoError(t, err) {
			continue
		}
		switch o := newobj.(type) {
		case *amb.ConsulResolver:
			resolvers = append(resolvers, o)
		case *amb.Mapping:
			mappings = append(mappings, consulMapping{Service: o.Spec.Service, Resolver: o.Spec.Resolver})
		case *amb.TCPMapping:
			mappings = append(mappings, consulMapping{Service: o.Spec.Service, Resolver: o.Spec.Resolver})
		}
	}

	assert.Equal(t, 1, len(resolvers))
	assert.Equal(t, 4, len(mappings))

	tw = &testWatcher{t: t, events: make(map[string]bool)}
	c = newConsulWatcher(tw.Watch)
	grp.Go("consul", c.run)
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

func (tw *testWatcher) Watch(ctx context.Context, resolver *amb.ConsulResolver, svc string, _ chan consulwatch.Endpoints) (Stopper, error) {
	rname := fmt.Sprintf("%s.%s", resolver.GetName(), resolver.GetNamespace())
	tw.Logf("%s:%s:watch", rname, svc)
	return &testStopper{watcher: tw, resolver: rname, service: svc}, nil
}

type testStopper struct {
	watcher  *testWatcher
	resolver string
	service  string
}

func (ts *testStopper) Stop() {
	ts.watcher.Logf("%s:%s:stop", ts.resolver, ts.service)
}
