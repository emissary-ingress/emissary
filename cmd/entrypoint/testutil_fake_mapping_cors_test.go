package entrypoint_test

import (
	"testing"

	"github.com/emissary-ingress/emissary/v3/cmd/entrypoint"
	apiv3_bootstrap "github.com/emissary-ingress/emissary/v3/pkg/api/envoy/config/bootstrap/v3"

	"github.com/stretchr/testify/assert"
)

func TestMappingCORSOriginsSlice(t *testing.T) {
	f := entrypoint.RunFake(t, entrypoint.FakeConfig{EnvoyConfig: true}, nil)
	err := f.UpsertYAML(`
---
apiVersion: getambassador.io/v3alpha1
kind: Listener
metadata:
  name: ambassador-listener-8080
  namespace: default
spec:
  port: 8080
  protocol: HTTP
  securityModel: XFP
  hostBinding:
    namespace:
      from: ALL
---
apiVersion: getambassador.io/v3alpha1
kind: Host
metadata:
  name: test-host
  namespace: default
spec:
  mappingSelector:
    matchLabels:
      host: minimal
---
apiVersion: getambassador.io/v3alpha1
kind: Mapping
metadata:
  name: foo
  namespace: default
  labels:
    host: minimal
spec:
  prefix: /foo
  service: foo.default
  cors:
    origins:
     - foo.example.com
     - bar.example.com
`)
	assert.NoError(t, err)

	err = f.Upsert(makeService("default", "foo"))
	assert.NoError(t, err)

	f.Flush()
	snap, err := f.GetSnapshot(HasMapping("default", "foo"))
	assert.NoError(t, err)
	assert.NotNil(t, snap)

	config, err := f.GetEnvoyConfig(func(config *apiv3_bootstrap.Bootstrap) bool {
		return FindCluster(config, ClusterNameContains("cluster_foo_default_default")) != nil
	})

	assert.NoError(t, err)

	listener := mustFindListenerByName(t, config, "ambassador-listener-8080")

	// Here we're looking for a route whose _action_ is to route to the cluster we want.
	routeAction := mustFindRouteActionToCluster(t, listener, "cluster_foo_default_default")
	assert.NotNil(t, routeAction.Cors)
	assert.Equal(t, len(routeAction.Cors.AllowOriginStringMatch), 2)
	for _, m := range routeAction.Cors.AllowOriginStringMatch {
		assert.Contains(t, []string{"bar.example.com", "foo.example.com"}, m.GetExact())

	}
}
