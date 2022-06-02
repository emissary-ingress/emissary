package entrypoint

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/datawire/ambassador/v2/pkg/snapshot/v1"
)

type subtest struct {
	name       string
	apiVersion string
}

// Predicate used to find snapshots that have the specified Secret
func hasSecret(name string, namespace string) func(snapshot *snapshot.Snapshot) bool {
	return func(snapshot *snapshot.Snapshot) bool {
		for _, s := range snapshot.Kubernetes.Secrets {
			if s.Name == name && s.Namespace == namespace {
				return true
			}
		}
		return false
	}
}

func TestFindFilterNamespacedSecret(t *testing.T) {
	subTests := []subtest{
		{"TestFindFilterNamespacedSecret V2", "getambassador.io/v2"},
		{"TestFindFilterNamespacedSecret V3alpha1", "getambassador.io/v3alpha1"},
	}

	for _, subTest := range subTests {
		t.Run(subTest.name, func(t *testing.T) {
			t.Setenv("EDGE_STACK", "true")
			f := RunFake(t, FakeConfig{EnvoyConfig: true, DiagdDebug: true}, nil)
			assert.NoError(t, f.UpsertYAML(`
---
apiVersion: `+subTest.apiVersion+`
kind: Filter
metadata:
  name: namespaced-secret-filter
  namespace: foo
spec:
  OAuth2:
    authorizationURL: https://login.example.com/dummy-client/v2
    clientID: dummy-id
    clientURL: https://dummy-client-url.com
    secretName: namespaced-secret
    secretNamespace: bar
---
apiVersion: v1
data:
  dummy-data: IGR1bW15LWRhdGE6IGZvbyAK
kind: Secret
metadata:
  name: namespaced-secret
  namespace: bar
type: Opaque
`))
			f.Flush()

			// After we flush the above config then secrets should be reconciled which should
			// result in the above secret that is referenced by our filter being added to the snapshot if everything succeeds
			snap, err := f.GetSnapshot(hasSecret("namespaced-secret", "bar"))
			require.NoError(t, err)
			assert.NotNil(t, snap)
			t.Setenv("EDGE_STACK", "")
		})
	}
}

func TestFindFilterSecretNoNamespace(t *testing.T) {
	subTests := []subtest{
		{"TestFindFilterSecretNoNamespace V2", "getambassador.io/v2"},
		{"TestFindFilterSecretNoNamespace V3alpha1", "getambassador.io/v3alpha1"},
	}

	for _, subTest := range subTests {
		t.Run(subTest.name, func(t *testing.T) {
			t.Setenv("EDGE_STACK", "true")
			f := RunFake(t, FakeConfig{EnvoyConfig: true, DiagdDebug: true}, nil)
			assert.NoError(t, f.UpsertYAML(`
---
apiVersion: `+subTest.apiVersion+`
kind: Filter
metadata:
  name: namespaced-secret-filter
  namespace: foo
spec:
  OAuth2:
    authorizationURL: https://login.example.com/dummy-client/v2
    clientID: dummy-id
    clientURL: https://dummy-client-url.com
    secretName: namespaced-secret
---
apiVersion: v1
data:
  dummy-data: IGR1bW15LWRhdGE6IGZvbyAK
kind: Secret
metadata:
  name: namespaced-secret
  namespace: foo
type: Opaque
`))
			f.Flush()

			// After we flush the above config then secrets should be reconciled which should
			// result in the above secret that is referenced by our filter being added to the snapshot if everything succeeds
			snap, err := f.GetSnapshot(hasSecret("namespaced-secret", "foo"))
			require.NoError(t, err)
			assert.NotNil(t, snap)
			t.Setenv("EDGE_STACK", "")
		})
	}
}
