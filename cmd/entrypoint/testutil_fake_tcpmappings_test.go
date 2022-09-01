package entrypoint_test

import (
	"testing"

	"github.com/emissary-ingress/emissary/v3/cmd/entrypoint"
	bootstrap "github.com/emissary-ingress/emissary/v3/pkg/api/envoy/config/bootstrap/v3"
	"github.com/emissary-ingress/emissary/v3/pkg/snapshot/v1"

	"github.com/stretchr/testify/assert"
)

func FirstSnapshot() func(snapshot *snapshot.Snapshot) bool {
	return func(snapshot *snapshot.Snapshot) bool {
		return true
	}
}

func TestFakeTCPMappings(t *testing.T) {
	f := entrypoint.RunFake(t, entrypoint.FakeConfig{EnvoyConfig: true, DiagdDebug: true}, nil)

	assert.NoError(t, f.UpsertFile("testdata/TCPMappings.yaml"))
	f.Flush()

	snap, err := f.GetSnapshot(FirstSnapshot())
	assert.NoError(t, err)
	assert.NotNil(t, snap)

	config, err := f.GetEnvoyConfig(func(config *bootstrap.Bootstrap) bool {
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
