package entrypoint_test

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/datawire/ambassador/v2/cmd/entrypoint"
	v3bootstrap "github.com/datawire/ambassador/v2/pkg/api/envoy/config/bootstrap/v3"
	v3 "github.com/datawire/ambassador/v2/pkg/api/envoy/type/v3"
	"github.com/datawire/ambassador/v2/pkg/kates"
	"github.com/datawire/ambassador/v2/pkg/snapshot/v1"
)

func AnySnapshot(_ *snapshot.Snapshot) bool {
	return true
}

func AnyConfig(_ *v3bootstrap.Bootstrap) bool {
	return true
}

func TestFake(t *testing.T) {
	f := entrypoint.RunFake(t, entrypoint.FakeConfig{EnvoyConfig: true}, nil)
	assert.NoError(t, f.UpsertFile("testdata/snapshot.yaml"))
	f.AutoFlush(true)

	snapshot, err := f.GetSnapshot(AnySnapshot)
	require.NoError(t, err)
	LogJSON(t, snapshot)

	envoyConfig, err := f.GetEnvoyConfig(AnyConfig)
	require.NoError(t, err)
	LogJSON(t, envoyConfig)

	assert.NoError(t, f.Delete("Mapping", "default", "foo"))

	snapshot, err = f.GetSnapshot(AnySnapshot)
	require.NoError(t, err)
	LogJSON(t, snapshot)

	envoyConfig, err = f.GetEnvoyConfig(AnyConfig)
	require.NoError(t, err)
	LogJSON(t, envoyConfig)

	/*f.ConsulEndpoints(endpointsBlob)
	f.ApplyFile()
	f.ApplyResources()
	f.Snapshot(snapshot1)
	f.Snapshot(snapshot2)
	f.Snapshot(snapshot3)
	f.Delete(namespace, name)
	f.Upsert(katesObject)
	f.UpsertString("kind: blah")*/

	// bluescape: create 50 hosts in different namespaces vs 50 hosts in the same namespace
	// consul data center other than dc1

}

func assertRoutePresent(t *testing.T, envoyConfig *v3bootstrap.Bootstrap, cluster string, weight int) {
	t.Helper()

	listener := mustFindListenerByName(t, envoyConfig, "ambassador-listener-8080")
	routes := mustFindRoutesToCluster(t, listener, cluster)

	for _, r := range routes {
		assert.Equal(t, uint32(weight), r.Match.RuntimeFraction.DefaultValue.Numerator)
		assert.Equal(t, v3.FractionalPercent_HUNDRED, r.Match.RuntimeFraction.DefaultValue.Denominator)
	}
}

func assertRouteNotPresent(t *testing.T, envoyConfig *v3bootstrap.Bootstrap, cluster string) {
	t.Helper()

	listener := mustFindListenerByName(t, envoyConfig, "ambassador-listener-8080")
	routes := findRoutesToCluster(listener, cluster)

	assert.Empty(t, routes)
}

func TestWeightWithCache(t *testing.T) {
	get_envoy_config := func(f *entrypoint.Fake, want_foo bool, want_bar bool) (*v3bootstrap.Bootstrap, error) {
		return f.GetEnvoyConfig(func(config *v3bootstrap.Bootstrap) bool {
			c_foo := FindCluster(config, ClusterNameContains("cluster_foo_"))
			c_bar := FindCluster(config, ClusterNameContains("cluster_bar_"))

			has_foo := c_foo != nil
			has_bar := c_bar != nil

			return (has_foo == want_foo) && (has_bar == want_bar)
		})
	}

	f := entrypoint.RunFake(t, entrypoint.FakeConfig{EnvoyConfig: true, DiagdDebug: false}, nil)
	assert.NoError(t, f.UpsertYAML(`
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
 hostname: foo.example.com
 requestPolicy:
  insecure:
   action: Route
---
apiVersion: getambassador.io/v3alpha1
kind: Mapping
metadata:
 name: mapping-foo
 namespace: default
 labels:
  host: minimal
spec:
 prefix: /foo/
 service: foo.default
`))

	f.Flush()

	// We need an Envoy config that has a foo cluster, but not a bar cluster.
	envoyConfig, err := get_envoy_config(f, true, false)
	require.NoError(t, err)
	assert.NotNil(t, envoyConfig)

	// Now we need to check the weights for our routes.
	assertRoutePresent(t, envoyConfig, "cluster_foo_default_default", 100)
	assertRouteNotPresent(t, envoyConfig, "cluster_bar_default_default")

	// Now add a bar mapping at weight 10.
	assert.NoError(t, f.UpsertYAML(`
---
apiVersion: getambassador.io/v3alpha1
kind: Mapping
metadata:
 name: mapping-bar
 namespace: default
 labels:
  host: minimal
spec:
 prefix: /foo/
 service: bar.default
 weight: 10
`))

	f.Flush()

	// We need an Envoy config that has a foo cluster and a bar cluster.
	envoyConfig, err = get_envoy_config(f, true, true)
	require.NoError(t, err)
	assert.NotNil(t, envoyConfig)

	// Check the weights in order: we should see the bar cluster at 10%, then the foo
	// cluster at 100%.
	assertRoutePresent(t, envoyConfig, "cluster_bar_default_default", 10)
	assertRoutePresent(t, envoyConfig, "cluster_foo_default_default", 100)

	// Now ramp the bar mapping to weight 50.
	assert.NoError(t, f.UpsertYAML(`
---
apiVersion: getambassador.io/v3alpha1
kind: Mapping
metadata:
 name: mapping-bar
 namespace: default
 labels:
  host: minimal
spec:
 prefix: /foo/
 service: bar.default
 weight: 50
`))

	f.Flush()

	// We need an Envoy config that has a foo cluster and a bar cluster.
	envoyConfig, err = get_envoy_config(f, true, true)
	require.NoError(t, err)
	assert.NotNil(t, envoyConfig)

	// Here we expect bar at 50%, then foo at 100%.
	assertRoutePresent(t, envoyConfig, "cluster_bar_default_default", 50)
	assertRoutePresent(t, envoyConfig, "cluster_foo_default_default", 100)

	assert.NoError(t, f.Delete("Mapping", "default", "mapping-foo"))
	f.Flush()

	// We need an Envoy config that has a bar cluster, but not a foo cluster...
	envoyConfig, err = get_envoy_config(f, false, true)
	require.NoError(t, err)
	assert.NotNil(t, envoyConfig)

	// ...and we should see the bar cluster at 100%.
	assertRoutePresent(t, envoyConfig, "cluster_bar_default_default", 100)
	assertRouteNotPresent(t, envoyConfig, "cluster_foo_default_default")

	// Now change bar's weight...
	assert.NoError(t, f.UpsertYAML(`
---
apiVersion: getambassador.io/v3alpha1
kind: Mapping
metadata:
 name: mapping-bar
 namespace: default
 labels:
  host: minimal
spec:
 prefix: /foo/
 service: bar.default
 weight: 20 
`))

	f.Flush()

	// ...and that should have absolutely no effect on what we see so far. We should
	// still see the bar cluster at 100% and no foo cluster.
	envoyConfig, err = get_envoy_config(f, false, true)
	require.NoError(t, err)
	assert.NotNil(t, envoyConfig)

	assertRoutePresent(t, envoyConfig, "cluster_bar_default_default", 100)
	assertRouteNotPresent(t, envoyConfig, "cluster_foo_default_default")

	// Now re-add the foo mapping.
	assert.NoError(t, f.UpsertYAML(`
---
apiVersion: getambassador.io/v3alpha1
kind: Mapping
metadata:
 name: mapping-foo
 namespace: default
 labels:
  host: minimal
spec:
 prefix: /foo/
 service: foo.default
`))

	f.Flush()

	// Now we should see both the foo cluster and the bar cluster...
	envoyConfig, err = get_envoy_config(f, true, true)
	require.NoError(t, err)
	assert.NotNil(t, envoyConfig)

	// ...and we should see the bar cluster drop to 20%, with the foo cluster now
	// at 100%.
	assertRoutePresent(t, envoyConfig, "cluster_bar_default_default", 20)
	assertRoutePresent(t, envoyConfig, "cluster_foo_default_default", 100)
}

func LogJSON(t testing.TB, obj interface{}) {
	t.Helper()
	bytes, err := json.MarshalIndent(obj, "", "  ")
	require.NoError(t, err)
	t.Log(string(bytes))
}

func TestFakeIstioCert(t *testing.T) {
	// Don't ask for the EnvoyConfig yet, 'cause we don't use it.
	f := entrypoint.RunFake(t, entrypoint.FakeConfig{EnvoyConfig: false}, nil)
	f.AutoFlush(true)

	assert.NoError(t, f.UpsertFile("testdata/tls-snap.yaml"))

	// t.Log(f.GetSnapshotString())

	snapshot, err := f.GetSnapshot(AnySnapshot)
	require.NoError(t, err)
	k := snapshot.Kubernetes

	if len(k.Secrets) != 1 {
		t.Errorf("needed 1 secret, got %d", len(k.Secrets))
	}

	istioSecret := kates.Secret{
		TypeMeta: kates.TypeMeta{
			APIVersion: "v1",
			Kind:       "Secret",
		},
		ObjectMeta: kates.ObjectMeta{
			Name:      "test-istio-secret",
			Namespace: "default",
		},
		Type: kates.SecretTypeTLS,
		Data: map[string][]byte{
			"tls.key": []byte("not-real-cert"),
			"tls.crt": []byte("not-real-pem"),
		},
	}

	f.SendIstioCertUpdate(entrypoint.IstioCertUpdate{
		Op:        "update",
		Name:      "test-istio-secret",
		Namespace: "default",
		Secret:    &istioSecret,
	})

	snapshot, err = f.GetSnapshot(AnySnapshot)
	require.NoError(t, err)
	k = snapshot.Kubernetes
	LogJSON(t, k)

	if len(k.Secrets) != 2 {
		t.Errorf("needed 2 secrets, got %d", len(k.Secrets))
	}
}
