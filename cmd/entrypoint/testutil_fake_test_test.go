package entrypoint_test

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/datawire/ambassador/v2/cmd/entrypoint"
	v3bootstrap "github.com/datawire/ambassador/v2/pkg/api/envoy/config/bootstrap/v3"
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
