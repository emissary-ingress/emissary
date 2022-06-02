package entrypoint

import (
	"testing"

	snapshotTypes "github.com/datawire/ambassador/v2/pkg/snapshot/v1"
	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
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
	err := findFilterSecret(testFilter, action)

	// We expect there to be an error for this test, but its just an error that gets logged
	// The main point of this case is to ensure that bogus inputs dont cause any panics
	assert.EqualError(t, err, `Filter object detected with a bogus "spec" field`)
	assert.Equal(t, 0, len(refs))
}

func TestFindFilterSecrets(t *testing.T) {
	t.Parallel()

	type subtest struct {
		inputSpec    map[string]interface{}
		expectedRefs []snapshotTypes.SecretRef
		expectedErr  string
	}
	// Iterate each test for all our API Versions
	apiVersions := []string{"getambassador.io/v2", "getambassador.io/v3alpha1"}
	subtests := map[string]subtest{
		"namespaced": {
			map[string]interface{}{
				"OAuth2": map[string]interface{}{
					"authorizationURL": "https://login.example.com/dummy-client/v2",
					"clientID":         "dummy-id",
					"clientURL":        "https://dummy-client-url.com",
					"secretName":       "namespaced-secret",
					"secretNamespace":  "bar",
				},
			},
			[]snapshotTypes.SecretRef{{Namespace: "bar", Name: "namespaced-secret"}},
			``,
		},
		"noNamespace": {
			map[string]interface{}{
				"OAuth2": map[string]interface{}{
					"authorizationURL": "https://login.example.com/dummy-client/v2",
					"clientID":         "dummy-id",
					"clientURL":        "https://dummy-client-url.com",
					"secretName":       "namespaced-secret",
				},
			},
			[]snapshotTypes.SecretRef{{Namespace: "foo", Name: "namespaced-secret"}},
			``,
		},
		"bogusOAuth": {
			map[string]interface{}{
				"OAuth2": true,
			},
			[]snapshotTypes.SecretRef{},
			`Filter object detected with a bogus "OAuth2" field`,
		},
		"bogusSecretName": {
			map[string]interface{}{
				"OAuth2": map[string]interface{}{
					"authorizationURL": "https://login.example.com/dummy-client/v2",
					"clientID":         "dummy-id",
					"clientURL":        "https://dummy-client-url.com",
					"secretName":       true,
				},
			},
			[]snapshotTypes.SecretRef{},
			`Filter object detected with a bogus "secretName" field`,
		},
		"bogusSecretNamespace": {
			map[string]interface{}{
				"OAuth2": map[string]interface{}{
					"authorizationURL": "https://login.example.com/dummy-client/v2",
					"clientID":         "dummy-id",
					"clientURL":        "https://dummy-client-url.com",
					"secretName":       "namespaced-secret",
					"secretNamespace":  true,
				},
			},
			[]snapshotTypes.SecretRef{},
			`Filter object detected with a bogus "secretNamespace" field`,
		},
		"noSecret": {
			map[string]interface{}{
				"OAuth2": map[string]interface{}{
					"authorizationURL": "https://login.example.com/dummy-client/v2",
					"clientID":         "dummy-id",
					"clientURL":        "https://dummy-client-url.com",
				},
			},
			[]snapshotTypes.SecretRef{},
			``,
		},
	}

	for _, apiVersion := range apiVersions {
		for name, subtest := range subtests {
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
				err := findFilterSecret(testFilter, action)
				// Some tests expect an error to be handled gracefully
				if subtest.expectedErr == `` {
					assert.NoError(t, err)
				} else {
					assert.EqualError(t, err, subtest.expectedErr)
				}
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
