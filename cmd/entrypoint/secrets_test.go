package entrypoint

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/emissary-ingress/emissary/v3/pkg/api/getambassador.io/v3alpha1"
	"github.com/emissary-ingress/emissary/v3/pkg/kates"
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
			expectedRefs: []snapshotTypes.SecretRef{
				{Namespace: "bar", Name: "namespaced-secret"},
			},
		},
		"noNamespace": {
			inputSpec: map[string]interface{}{
				"OAuth2": map[string]interface{}{
					"authorizationURL": "https://login.example.com/dummy-client/v2",
					"clientID":         "dummy-id",
					"clientURL":        "https://dummy-client-url.com",
					"secretName":       "namespaced-secret",
				},
				"APIKey": map[string]interface{}{
					"http_header": "x-api-key",
					"keys": []interface{}{
						map[string]interface{}{
							"value": "super-secret",
						},
						map[string]interface{}{
							"secretName": "namespaced-secret-api",
						},
						map[string]interface{}{
							"secretName":      "namespaced-secret-api-2",
							"secretNamespace": "",
						},
					},
				},
			},
			expectedRefs: []snapshotTypes.SecretRef{
				{Namespace: "foo", Name: "namespaced-secret"},
				{Namespace: "foo", Name: "namespaced-secret-api"},
				{Namespace: "foo", Name: "namespaced-secret-api-2"},
			},
		},
		"bogus": {
			inputSpec: map[string]interface{}{
				"OAuth2": true,
				"APIKey": true,
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
				"APIKey": map[string]interface{}{
					"http_header": "x-api-key",
					"keys": []interface{}{
						map[string]interface{}{
							"value": "super-secret",
						},
						map[string]interface{}{
							"secretName": true,
						},
					},
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
				"APIKey": map[string]interface{}{
					"http_header": "x-api-key",
					"keys": []interface{}{
						map[string]interface{}{
							"value": "super-secret",
						},
						map[string]interface{}{},
					},
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

func TestReconcileSecrets(t *testing.T) {

	type testCase struct {
		Description  string `json:"description"`
		AmbassadorID string `json:"ambassadorId"`
		IsEdgeStack  bool   `json:"isEdgeStack"`
		Input        struct {
			Services    []*kates.Service             `json:"services"`
			Ingresses   []*snapshotTypes.Ingress     `json:"ingresses"`
			K8Secrets   []*corev1.Secret             `json:"k8sSecrets"`
			Hosts       []*v3alpha1.Host             `json:"hosts"`
			TLSContexts []*v3alpha1.TLSContext       `json:"tlsContexts"`
			Modules     []*v3alpha1.Module           `json:"modules"`
			Filters     []*unstructured.Unstructured `json:"filters"`
		} `json:"input"`
		Expected struct {
			Secrets []*corev1.Secret `json:"secrets"`
		} `json:"expected"`
	}

	testdataBaseDir := "./testdata/reconcile-secrets"
	testcases := loadTestCases[testCase](t, testdataBaseDir, "*")

	for _, tc := range testcases {
		t.Run(tc.Description, func(t *testing.T) {
			ctx := context.Background()

			ambMetaInfo := &snapshotTypes.AmbassadorMetaInfo{
				AmbassadorID: tc.AmbassadorID,
			}

			if tc.AmbassadorID == "" {
				ambMetaInfo.AmbassadorID = "default"
			}

			if tc.IsEdgeStack {
				t.Setenv("EDGE_STACK", "true")
			}

			sh, err := NewSnapshotHolder(ambMetaInfo)
			require.NoError(t, err)

			sh.k8sSnapshot = NewKubernetesSnapshot()

			sh.k8sSnapshot.Services = tc.Input.Services
			sh.k8sSnapshot.Ingresses = tc.Input.Ingresses
			sh.k8sSnapshot.K8sSecrets = tc.Input.K8Secrets
			sh.k8sSnapshot.Hosts = tc.Input.Hosts
			sh.k8sSnapshot.TLSContexts = tc.Input.TLSContexts
			sh.k8sSnapshot.Modules = tc.Input.Modules
			sh.k8sSnapshot.Filters = tc.Input.Filters

			err = sh.k8sSnapshot.PopulateAnnotations(ctx)
			require.NoError(t, err)

			err = ReconcileSecrets(ctx, sh)
			require.NoError(t, err)

			// validate secrets match
			expectedReconciledSecretCount := len(tc.Expected.Secrets)
			actualReconciledSecretCount := len(sh.k8sSnapshot.Secrets)
			require.Equal(t, expectedReconciledSecretCount, actualReconciledSecretCount,
				fmt.Sprintf("expected sh.k8sSnapshot.Secrets to have %d but reconciled %d secrets", expectedReconciledSecretCount, actualReconciledSecretCount),
			)

			assert.ElementsMatch(t, tc.Expected.Secrets, sh.k8sSnapshot.Secrets)
		})
	}
}
