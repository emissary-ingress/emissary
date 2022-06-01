package entrypoint_test

import (
	"testing"

	"github.com/datawire/ambassador/v2/cmd/entrypoint"
	"github.com/datawire/ambassador/v2/pkg/snapshot/v1"
	snapshotTypes "github.com/datawire/ambassador/v2/pkg/snapshot/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// Predicate used to find snapshots that have the specified Secret
func HasSecret(name string, namespace string) func(snapshot *snapshot.Snapshot) bool {
	return func(snapshot *snapshot.Snapshot) bool {
		for _, s := range snapshot.Kubernetes.Secrets {
			if s.Name == name && s.Namespace == namespace {
				return true
			}
		}
		return false
	}
}

func TestFindFilterNamespacedSecretV3alpha1(t *testing.T) {
	t.Setenv("EDGE_STACK", "true")
	f := entrypoint.RunFake(t, entrypoint.FakeConfig{EnvoyConfig: true, DiagdDebug: true}, nil)

	assert.NoError(t, f.UpsertYAML(`
---
apiVersion: getambassador.io/v3alpha1
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
	snap, err := f.GetSnapshot(HasSecret("namespaced-secret", "bar"))
	require.NoError(t, err)
	assert.NotNil(t, snap)
	t.Setenv("EDGE_STACK", "")
}

func TestFindFilterNamespacedSecretV2(t *testing.T) {
	t.Setenv("EDGE_STACK", "true")
	f := entrypoint.RunFake(t, entrypoint.FakeConfig{EnvoyConfig: true, DiagdDebug: true}, nil)

	assert.NoError(t, f.UpsertYAML(`
---
apiVersion: getambassador.io/v2
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
	snap, err := f.GetSnapshot(HasSecret("namespaced-secret", "foo"))
	require.NoError(t, err)
	assert.NotNil(t, snap)
	t.Setenv("EDGE_STACK", "")
}

func TestFindFilterSecretNamespaceV2(t *testing.T) {
	t.Setenv("EDGE_STACK", "true")
	refs := map[snapshotTypes.SecretRef]bool{}
	action := func(ref snapshotTypes.SecretRef) {
		refs[ref] = true
	}

	testFilter := &unstructured.Unstructured{
		Object: map[string]interface{}{

			"apiVersion": "getambassador.io/v2",
			"kind":       "Filter",
			"metadata": map[string]interface{}{
				"name":      "namespaced-secret-filter",
				"namespace": "foo",
			},
			"spec": map[string]interface{}{
				"OAuth2": map[string]interface{}{
					"authorizationURL": "https://login.example.com/dummy-client/v2",
					"clientID":         "dummy-id",
					"clientURL":        "https://dummy-client-url.com",
					"secretName":       "namespaced-secret",
					"secretNamespace":  "bar",
				},
			},
		},
	}

	assert.NoError(t, entrypoint.FindFilterSecret(testFilter, action))

	assert.Equal(t, 1, len(refs))
	expectedRef := snapshotTypes.SecretRef{Namespace: "bar", Name: "namespaced-secret"}

	// Test that we properly processed the secret reference for the dummy filter
	assert.True(t, refs[expectedRef])
	t.Setenv("EDGE_STACK", "")
}

func TestFindFilterSecretNamespaceV3alpha1(t *testing.T) {
	t.Setenv("EDGE_STACK", "true")
	refs := map[snapshotTypes.SecretRef]bool{}
	action := func(ref snapshotTypes.SecretRef) {
		refs[ref] = true
	}

	testFilter := &unstructured.Unstructured{
		Object: map[string]interface{}{

			"apiVersion": "getambassador.io/v3alpha1",
			"kind":       "Filter",
			"metadata": map[string]interface{}{
				"name":      "namespaced-secret-filter",
				"namespace": "foo",
			},
			"spec": map[string]interface{}{
				"OAuth2": map[string]interface{}{
					"authorizationURL": "https://login.example.com/dummy-client/v2",
					"clientID":         "dummy-id",
					"clientURL":        "https://dummy-client-url.com",
					"secretName":       "namespaced-secret",
					"secretNamespace":  "bar",
				},
			},
		},
	}

	assert.NoError(t, entrypoint.FindFilterSecret(testFilter, action))

	assert.Equal(t, 1, len(refs))
	expectedRef := snapshotTypes.SecretRef{Namespace: "bar", Name: "namespaced-secret"}

	// Test that we properly processed the secret reference for the dummy filter
	assert.True(t, refs[expectedRef])
	t.Setenv("EDGE_STACK", "")
}

// Tests whether we successfully process the secretreference for the secret needef by the provided filter.
// The secret does not specify a namespace so it should take the namepsace of the Filter.
func TestFindFilterSecretNoNamespace(t *testing.T) {
	t.Setenv("EDGE_STACK", "true")
	refs := map[snapshotTypes.SecretRef]bool{}
	action := func(ref snapshotTypes.SecretRef) {
		refs[ref] = true
	}

	testFilter := &unstructured.Unstructured{
		Object: map[string]interface{}{

			"apiVersion": "getambassador.io/v2",
			"kind":       "Filter",
			"metadata": map[string]interface{}{
				"name":      "namespaced-secret-filter",
				"namespace": "foo",
			},
			"spec": map[string]interface{}{
				"OAuth2": map[string]interface{}{
					"authorizationURL": "https://login.example.com/dummy-client/v2",
					"clientID":         "dummy-id",
					"clientURL":        "https://dummy-client-url.com",
					"secretName":       "namespaced-secret",
				},
			},
		},
	}

	assert.NoError(t, entrypoint.FindFilterSecret(testFilter, action))
	assert.Equal(t, 1, len(refs))
	expectedRef := snapshotTypes.SecretRef{Namespace: "foo", Name: "namespaced-secret"}

	// Test that we properly processed the secret reference for the dummy filter
	assert.True(t, refs[expectedRef])
	t.Setenv("EDGE_STACK", "")
}

// Tests whether providing a Filter with a bogus spec
func TestFindFilterSecretBogus(t *testing.T) {
	t.Setenv("EDGE_STACK", "true")
	refs := map[snapshotTypes.SecretRef]bool{}
	action := func(ref snapshotTypes.SecretRef) {
		refs[ref] = true
	}

	testFilter := &unstructured.Unstructured{
		Object: map[string]interface{}{

			"apiVersion": "getambassador.io/v2",
			"kind":       "Filter",
			"metadata": map[string]interface{}{
				"name":      "namespaced-secret-filter",
				"namespace": "foo",
			},
			"spec": "bogus",
		},
	}
	// We shouldnt process any secret refs from this bogus Filter
	err := entrypoint.FindFilterSecret(testFilter, action)

	// We expect there to be an error for this test, but its just an error that gets logged
	// The main point of this case is to ensure that bogus inputs dont cause any panics
	assert.NotNil(t, err)
	assert.Equal(t, 0, len(refs))
	t.Setenv("EDGE_STACK", "")
}
