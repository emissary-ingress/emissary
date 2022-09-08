package entrypoint_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	snapshot "github.com/datawire/ambassador/v2/pkg/snapshot/v1"

	"github.com/datawire/ambassador/v2/cmd/entrypoint"
	bootstrap "github.com/datawire/ambassador/v2/pkg/api/envoy/config/bootstrap/v3"
	v3tcpproxy "github.com/datawire/ambassador/v2/pkg/api/envoy/extensions/filters/network/tcp_proxy/v3"
)

func FirstSnapshot() func(snapshot *snapshot.Snapshot) bool {
	return func(snapshot *snapshot.Snapshot) bool {
		return true
	}
}

func BothOf(pred1, pred2 func(snapshot *snapshot.Snapshot) bool) func(snapshot *snapshot.Snapshot) bool {
	return func(snapshot *snapshot.Snapshot) bool {
		return pred1(snapshot) && pred2(snapshot)
	}
}

func HasTCPMapping(namespace, name string) func(snapshot *snapshot.Snapshot) bool {
	return func(snapshot *snapshot.Snapshot) bool {
		for _, m := range snapshot.Kubernetes.TCPMappings {
			if m.Namespace == namespace && m.Name == name {
				return true
			}
		}

		return false
	}
}

func TestFakeTCPMappings(t *testing.T) {
	f := entrypoint.RunFake(t, entrypoint.FakeConfig{EnvoyConfig: true, DiagdDebug: true}, nil)

	assert.NoError(t, f.UpsertFile("testdata/TCPMappings.yaml"))
	f.Flush()

	snap, err := f.GetSnapshot(BothOf(HasTCPMapping("default", "tcpmapping-foo"), HasTCPMapping("default", "tcpmapping-bar")))
	assert.NoError(t, err)
	assert.NotNil(t, snap)

	// We must have a listener named "ambassador-listener-3306".
	config, err := f.GetEnvoyConfig(func(config *bootstrap.Bootstrap) bool {
		return findListenerByName(config, "ambassador-listener-3306") != nil
	})

	assert.NoError(t, err)
	assert.NotNil(t, config)

	listener := mustFindListenerByName(t, config, "ambassador-listener-3306")

	// We need two chains in the listener.
	assert.Equal(t, 2, len(listener.FilterChains))

	// One chain must match server_name foo.example.com, the other bar.example.com.
	// The foo.example.com chain must use cluster cluster_mysql01_3306_default; the
	// bar.example.com chain must use cluster cluster_mysql02_3306_default.

	clusterNames := map[string]string{
		"foo.example.com": "cluster_mysql01_3306_default",
		"bar.example.com": "cluster_mysql02_3306_default",
	}

	found := map[string]bool{}

	for _, chain := range listener.FilterChains {
		match := chain.FilterChainMatch

		assert.NotNil(t, match)
		assert.NotNil(t, match.ServerNames)
		assert.Equal(t, 1, len(match.ServerNames))

		serverName := match.ServerNames[0]
		found[serverName] = true

		// Additional, each chain must have a single filter, which must be a V3 TCP
		// proxy.

		assert.NotNil(t, chain.Filters)
		assert.Equal(t, 1, len(chain.Filters))
		assert.Equal(t, "envoy.filters.network.tcp_proxy", chain.Filters[0].Name)

		typedConfig := chain.Filters[0].GetTypedConfig()
		assert.NotNil(t, typedConfig)

		tcpProxy := &v3tcpproxy.TcpProxy{}
		assert.NoError(t, typedConfig.UnmarshalTo(tcpProxy))

		// That TCP proxy must have a single cluster, which must be named as described
		// above.
		clusterSpec := tcpProxy.GetClusterSpecifier().(*v3tcpproxy.TcpProxy_WeightedClusters)
		assert.NotNil(t, clusterSpec)

		clusters := clusterSpec.WeightedClusters.GetClusters()

		assert.Equal(t, 1, len(clusters))
		assert.Equal(t, clusterNames[serverName], clusters[0].GetName())

		// The cluster must also have weight 100. (Yes, we have to explicitly check
		// for a uint32 value of 100. Sigh.)
		assert.Equal(t, uint32(100), clusters[0].GetWeight())

		// Both filters need a stat_prefix for "ingress_tls_3306".
		assert.Equal(t, "ingress_tls_3306", tcpProxy.GetStatPrefix())
	}

	assert.True(t, found["foo.example.com"])
	assert.True(t, found["bar.example.com"])
}
