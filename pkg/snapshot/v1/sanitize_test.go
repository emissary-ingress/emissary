package snapshot_test

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/datawire/ambassador/v2/pkg/kates/k8sresourcetypes"
	snapshotTypes "github.com/datawire/ambassador/v2/pkg/snapshot/v1"
)

func getUnstructured(objStr string) *k8sresourcetypes.Unstructured {
	var obj map[string]interface{}
	_ = json.Unmarshal([]byte(objStr), &obj)
	unstructured := &k8sresourcetypes.Unstructured{}
	unstructured.SetUnstructuredContent(obj)
	return unstructured
}

var sanitizeTests = []struct {
	testName          string
	unsanitized       *snapshotTypes.Snapshot
	expectedSanitized *snapshotTypes.Snapshot
}{
	{
		testName: "secrets",
		unsanitized: &snapshotTypes.Snapshot{
			Kubernetes: &snapshotTypes.KubernetesSnapshot{
				Secrets: []*k8sresourcetypes.Secret{
					{},
					{
						TypeMeta: metav1.TypeMeta{
							Kind:       "Secret",
							APIVersion: "v1",
						},
						ObjectMeta: metav1.ObjectMeta{
							Name:            "secret-1",
							Namespace:       "ns",
							ResourceVersion: "resourceversion",
							Labels:          map[string]string{"label": "unset"},
							Annotations:     map[string]string{"also": "unset"},
						},
						Type: "Opaque",
						Data: map[string][]byte{
							"data1": []byte("blahblahblah"),
							"data2": []byte("otherblahblahblah"),
						},
					},
					{
						TypeMeta: metav1.TypeMeta{
							Kind:       "Secret",
							APIVersion: "v1",
						},
						ObjectMeta: metav1.ObjectMeta{
							Name:            "other-secret",
							Namespace:       "other-ns",
							ResourceVersion: "resourceversion",
							Labels:          map[string]string{"label": "unset"},
							Annotations:     map[string]string{"also": "unset"},
						},
						Type: "kubernetes.io/tls",
						Data: map[string][]byte{
							"data3": []byte("bleepblorp"),
							"data4": []byte("realsecret"),
						},
					},
				},
			},
		},
		expectedSanitized: &snapshotTypes.Snapshot{
			Kubernetes: &snapshotTypes.KubernetesSnapshot{
				Secrets: []*k8sresourcetypes.Secret{
					{
						Data: map[string][]byte{},
					},
					{
						TypeMeta: metav1.TypeMeta{
							Kind:       "Secret",
							APIVersion: "v1",
						},
						ObjectMeta: metav1.ObjectMeta{
							Name:      "secret-1",
							Namespace: "ns",
						},
						Type: "Opaque",
						Data: map[string][]byte{
							"data1": []byte("<REDACTED>"),
							"data2": []byte("<REDACTED>"),
						},
					},
					{
						TypeMeta: metav1.TypeMeta{
							Kind:       "Secret",
							APIVersion: "v1",
						},
						ObjectMeta: metav1.ObjectMeta{
							Name:      "other-secret",
							Namespace: "other-ns",
						},
						Type: "kubernetes.io/tls",
						Data: map[string][]byte{
							"data3": []byte("<REDACTED>"),
							"data4": []byte("<REDACTED>"),
						},
					},
				},
			},
		},
	},
	{
		testName: "invalid",
		unsanitized: &snapshotTypes.Snapshot{
			Invalid: []*k8sresourcetypes.Unstructured{
				getUnstructured(`
                                        {
                                                "kind":"WeirdKind",
                                                "apiVersion":"v1",
                                                "metadata": {
                                                        "name":"hi",
                                                        "namespace":"default"
                                                },
                                                "errors": "someerrors",
                                                "wat":"dontshowthis"
                                        }`),
				getUnstructured(`{}`),
			},
		},
		expectedSanitized: &snapshotTypes.Snapshot{
			Invalid: []*k8sresourcetypes.Unstructured{
				getUnstructured(`
                                        {
                                                "kind":"WeirdKind",
                                                "apiVersion":"v1",
                                                "metadata": {
                                                        "name":"hi",
                                                        "namespace":"default"
                                                },
                                                "errors":"someerrors"
                                        }`),
				getUnstructured(`{"apiVersion":"","kind":""}`),
			},
		},
	},
	{
		testName:          "empty",
		unsanitized:       &snapshotTypes.Snapshot{},
		expectedSanitized: &snapshotTypes.Snapshot{},
	},
}

func TestSanitize(t *testing.T) {
	for _, sanitizeTest := range sanitizeTests {
		t.Run(sanitizeTest.testName, func(innerT *testing.T) {
			snapshot := *sanitizeTest.unsanitized
			expected := *sanitizeTest.expectedSanitized

			err := snapshot.Sanitize()

			assert.Nil(innerT, err)
			assert.Equal(innerT, expected, snapshot)
		})
	}
}
