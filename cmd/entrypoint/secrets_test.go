package entrypoint

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	snapshotTypes "github.com/emissary-ingress/emissary/v3/pkg/snapshot/v1"
)

// Tests whether providing a Filter with a bogus spec
// This is outside the table since we're providing a bogus spec and not the otherwise expected interface
func TestFindFilterSecretBogus(t *testing.T) {
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
	assert.NoError(t, findFilterSecret(testFilter, action))
	assert.Equal(t, 0, len(refs))
}

func TestFindFilterSecrets(t *testing.T) {
	t.Parallel()

	type subtest struct {
		inputSpec    map[string]interface{}
		expectedRefs []snapshotTypes.SecretRef
	}
	// Iterate each test for all our API Versions
	apiVersions := []string{"getambassador.io/v2", "getambassador.io/v3alpha1"}
	subtests := map[string]subtest{
		"namespaced": {
			inputSpec: map[string]interface{}{
				"OAuth2": map[string]interface{}{
					"authorizationURL": "https://login.example.com/dummy-client/v2",
					"clientID":         "dummy-id",
					"clientURL":        "https://dummy-client-url.com",
					"secretName":       "namespaced-secret",
					"secretNamespace":  "bar",
				},
			},
			expectedRefs: []snapshotTypes.SecretRef{{Namespace: "bar", Name: "namespaced-secret"}},
		},
		"noNamespace": {
			inputSpec: map[string]interface{}{
				"OAuth2": map[string]interface{}{
					"authorizationURL": "https://login.example.com/dummy-client/v2",
					"clientID":         "dummy-id",
					"clientURL":        "https://dummy-client-url.com",
					"secretName":       "namespaced-secret",
				},
			},
			expectedRefs: []snapshotTypes.SecretRef{{Namespace: "foo", Name: "namespaced-secret"}},
		},
		"bogusOAuth": {
			inputSpec: map[string]interface{}{
				"OAuth2": true,
			},
			expectedRefs: []snapshotTypes.SecretRef{},
		},
		"bogusSecretName": {
			inputSpec: map[string]interface{}{
				"OAuth2": map[string]interface{}{
					"authorizationURL": "https://login.example.com/dummy-client/v2",
					"clientID":         "dummy-id",
					"clientURL":        "https://dummy-client-url.com",
					"secretName":       true,
				},
			},
			expectedRefs: []snapshotTypes.SecretRef{},
		},
		"bogusSecretNamespace": {
			inputSpec: map[string]interface{}{
				"OAuth2": map[string]interface{}{
					"authorizationURL": "https://login.example.com/dummy-client/v2",
					"clientID":         "dummy-id",
					"clientURL":        "https://dummy-client-url.com",
					"secretName":       "namespaced-secret",
					"secretNamespace":  true,
				},
			},
			expectedRefs: []snapshotTypes.SecretRef{},
		},
		"noSecret": {
			inputSpec: map[string]interface{}{
				"OAuth2": map[string]interface{}{
					"authorizationURL": "https://login.example.com/dummy-client/v2",
					"clientID":         "dummy-id",
					"clientURL":        "https://dummy-client-url.com",
				},
			},
			expectedRefs: []snapshotTypes.SecretRef{},
		},
	}

	for _, apiVersion := range apiVersions {
		for name, subtest := range subtests {
			subtest := subtest // capture loop variable
			t.Run(name, func(t *testing.T) {
				t.Parallel()
				testFilter := &unstructured.Unstructured{
					Object: map[string]interface{}{
						"apiVersion": apiVersion,
						"kind":       "Filter",
						"metadata": map[string]interface{}{
							"name":      "namespaced-secret-filter",
							"namespace": "foo",
						},
						"spec": subtest.inputSpec,
					},
				}
				refs := map[snapshotTypes.SecretRef]bool{}
				action := func(ref snapshotTypes.SecretRef) {
					refs[ref] = true
				}
				assert.NoError(t, findFilterSecret(testFilter, action))
				// Check if we got the right number of secret references and that nothing weird happened
				assert.Equal(t, len(subtest.expectedRefs), len(refs))
				// Make sure any expected secret references exist
				for _, ref := range subtest.expectedRefs {
					assert.True(t, refs[ref])
				}
			})
		}
	}
}
