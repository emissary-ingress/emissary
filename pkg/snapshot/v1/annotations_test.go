package snapshot_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/datawire/dlib/dlog"
	amb "github.com/emissary-ingress/emissary/v3/pkg/api/getambassador.io/v3alpha1"
	"github.com/emissary-ingress/emissary/v3/pkg/kates"
	"github.com/emissary-ingress/emissary/v3/pkg/kates/k8s_resource_types"
	snapshotTypes "github.com/emissary-ingress/emissary/v3/pkg/snapshot/v1"
)

func getModuleSpec(t *testing.T, rawconfig string) amb.UntypedDict {
	moduleConfig := amb.UntypedDict{}
	if err := json.Unmarshal([]byte(rawconfig), &moduleConfig); err != nil {
		t.Fatal(t)
	}
	return moduleConfig
}

func TestParseAnnotations(t *testing.T) {
	mapping := `
---
apiVersion: getambassador.io/v3alpha1
kind: Mapping
name: quote-backend
prefix: /backend/
service: quote:80
`

	svc := &kates.Service{
		TypeMeta: metav1.TypeMeta{
			Kind: "Service",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "svc",
			Namespace: "ambassador",
			Annotations: map[string]string{
				"getambassador.io/config": mapping,
			},
		},
	}

	svcWithEmptyAnnotation := &kates.Service{
		TypeMeta: metav1.TypeMeta{
			Kind: "Service",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "svc-empty",
			Namespace: "ambassador",
			Annotations: map[string]string{
				"getambassador.io/config": "",
			},
		},
	}

	svcWithMissingAnnotation := &kates.Service{
		TypeMeta: metav1.TypeMeta{
			Kind: "Service",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:        "svc-missing",
			Namespace:   "ambassador",
			Annotations: map[string]string{},
		},
	}

	ingHost := `
---
apiVersion: getambassador.io/v3alpha1
kind: Mapping
name: cool-mapping
prefix: /blah/
`

	ingress := &k8s_resource_types.Ingress{
		TypeMeta: metav1.TypeMeta{
			Kind: "Ingress",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "ingress",
			Namespace: "somens",
			Annotations: map[string]string{
				"getambassador.io/config": ingHost,
			},
		},
	}

	ambSvcAnnotations := `
---
apiVersion: getambassador.io/v3alpha1
kind: Module
name: ambassador
config:
  diagnostics:
    enabled: true
---
apiVersion: getambassador.io/v3alpha1
kind: KubernetesEndpointResolver
name: endpoint`

	ambSvc := &kates.Service{
		TypeMeta: metav1.TypeMeta{
			Kind: "Service",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "ambassador",
			Namespace: "ambassador",
			Annotations: map[string]string{
				"getambassador.io/config": ambSvcAnnotations,
			},
		},
	}

	unparsedAnnotation := `
---
apiVersion: getambassador.io/v3alpha1
kind: Mapping
name: dont-parse
prefix: /blah/`

	ignoredHost := &amb.Host{
		TypeMeta: metav1.TypeMeta{
			Kind: "Host",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "ambassador",
			Namespace: "ambassador",
			Annotations: map[string]string{
				"getambassador.io/config": unparsedAnnotation,
			},
		},
	}

	ks := &snapshotTypes.KubernetesSnapshot{
		Services:  []*kates.Service{svc, ambSvc, svcWithEmptyAnnotation, svcWithMissingAnnotation},
		Ingresses: []*snapshotTypes.Ingress{{Ingress: *ingress}},
		Hosts:     []*amb.Host{ignoredHost},
	}

	ctx := dlog.NewTestContext(t, false)

	err := ks.PopulateAnnotations(ctx)
	assert.NoError(t, err)
	assert.Equal(t, len(ks.Services), 4)
	assert.Equal(t, map[string]snapshotTypes.AnnotationList{
		"Service/svc.ambassador": {
			&amb.Mapping{
				TypeMeta: metav1.TypeMeta{
					Kind:       "Mapping",
					APIVersion: "getambassador.io/v3alpha1",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "quote-backend",
					Namespace: "ambassador",
				},
				Spec: amb.MappingSpec{
					Prefix:  "/backend/",
					Service: "quote:80",
				},
			},
		},
		"Ingress/ingress.somens": {
			&kates.Unstructured{
				Object: map[string]interface{}{
					"apiVersion": "getambassador.io/v3alpha1",
					"kind":       "Mapping",
					"metadata": map[string]interface{}{
						"name":      "cool-mapping",
						"namespace": "somens",
					},
					"spec": map[string]interface{}{
						"prefix": "/blah/",
					},
					"errors": "spec.service in body is required",
				},
			},
		},
		"Service/ambassador.ambassador": {
			&amb.Module{
				TypeMeta: metav1.TypeMeta{
					Kind:       "Module",
					APIVersion: "getambassador.io/v3alpha1",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "ambassador",
					Namespace: "ambassador",
				},
				Spec: amb.ModuleSpec{
					Config: getModuleSpec(t, `{"diagnostics":{"enabled":true}}`),
				},
			},
			&amb.KubernetesEndpointResolver{
				TypeMeta: metav1.TypeMeta{
					Kind:       "KubernetesEndpointResolver",
					APIVersion: "getambassador.io/v3alpha1",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "endpoint",
					Namespace: "ambassador",
				},
			},
		},
	}, ks.Annotations)
}

func TestConvertAnnotation(t *testing.T) {
	testcases := map[string]struct {
		inputString       string
		inputParentNS     string
		inputParentLabels map[string]string

		outputObj kates.Object
	}{
		"mapping": {
			inputString: `
---
apiVersion: getambassador.io/v3alpha1
kind: Mapping
name: cool-mapping
prefix: /blah/
service: quote:80`,
			inputParentNS:     "somens",
			inputParentLabels: map[string]string{},
			outputObj: &amb.Mapping{
				TypeMeta: metav1.TypeMeta{
					Kind:       "Mapping",
					APIVersion: "getambassador.io/v3alpha1",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "cool-mapping",
					Namespace: "somens",
					Labels:    map[string]string{},
				},
				Spec: amb.MappingSpec{
					Prefix:  "/blah/",
					Service: "quote:80",
				},
			},
		},
		"old-group-v0": {
			inputString: `
---
apiVersion: getambassador.io/v3alpha1
kind: Mapping
name: cool-mapping
prefix: /blah/
service: quote:80`,
			inputParentNS:     "somens",
			inputParentLabels: map[string]string{},
			outputObj: &amb.Mapping{
				TypeMeta: metav1.TypeMeta{
					Kind:       "Mapping",
					APIVersion: "getambassador.io/v3alpha1",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "cool-mapping",
					Namespace: "somens",
					Labels:    map[string]string{},
				},
				Spec: amb.MappingSpec{
					Prefix:  "/blah/",
					Service: "quote:80",
				},
			},
		},
		"old-group-v1": {
			inputString: `
---
apiVersion: getambassador.io/v3alpha1
kind: Mapping
name: cool-mapping
prefix: /blah/
service: quote:80`,
			inputParentNS:     "somens",
			inputParentLabels: map[string]string{},
			outputObj: &amb.Mapping{
				TypeMeta: metav1.TypeMeta{
					Kind:       "Mapping",
					APIVersion: "getambassador.io/v3alpha1",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "cool-mapping",
					Namespace: "somens",
					Labels:    map[string]string{},
				},
				Spec: amb.MappingSpec{
					Prefix:  "/blah/",
					Service: "quote:80",
				},
			},
		},
		"label-override": {
			inputString: `
---
apiVersion: getambassador.io/v3alpha1
kind: Mapping
name: cool-mapping
metadata_labels:
  bleep: blorp
prefix: /blah/
service: quote:80`,
			inputParentNS:     "somens",
			inputParentLabels: map[string]string{"should": "override"},
			outputObj: &amb.Mapping{
				TypeMeta: metav1.TypeMeta{
					Kind:       "Mapping",
					APIVersion: "getambassador.io/v3alpha1",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "cool-mapping",
					Namespace: "somens",
					Labels: map[string]string{
						"bleep": "blorp",
					},
				},
				Spec: amb.MappingSpec{
					Prefix:  "/blah/",
					Service: "quote:80",
				},
			},
		},
		"parent-labels": {
			inputString: `
---
apiVersion: getambassador.io/v3alpha1
kind: Mapping
name: cool-mapping
prefix: /blah/
service: quote:80`,
			inputParentNS: "somens",
			inputParentLabels: map[string]string{
				"use": "theselabels",
			},
			outputObj: &amb.Mapping{
				TypeMeta: metav1.TypeMeta{
					Kind:       "Mapping",
					APIVersion: "getambassador.io/v3alpha1",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "cool-mapping",
					Namespace: "somens",
					Labels: map[string]string{
						"use": "theselabels",
					},
				},
				Spec: amb.MappingSpec{
					Prefix:  "/blah/",
					Service: "quote:80",
				},
			},
		},
		"module": {
			inputString: `
---
apiVersion: getambassador.io/v3alpha1
kind: Module
name: ambassador
config:
  diagnostics:
    enabled: true`,
			inputParentNS:     "somens",
			inputParentLabels: map[string]string{},
			outputObj: &amb.Module{
				TypeMeta: metav1.TypeMeta{
					Kind:       "Module",
					APIVersion: "getambassador.io/v3alpha1",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "ambassador",
					Namespace: "somens",
					Labels:    map[string]string{},
				},
				Spec: amb.ModuleSpec{
					Config: getModuleSpec(t, `{"diagnostics":{"enabled":true}}`),
				},
			},
		},
	}

	for tcName, tc := range testcases {
		tc := tc
		t.Run(tcName, func(t *testing.T) {
			parentObj := &kates.Unstructured{
				Object: map[string]interface{}{
					"apiVersion": "v1",
					"kind":       "Service",
					"metadata": map[string]interface{}{
						"name":      "parentname",
						"namespace": tc.inputParentNS,
						"labels": func() map[string]interface{} {
							ret := make(map[string]interface{}, len(tc.inputParentLabels))
							for k, v := range tc.inputParentLabels {
								ret[k] = v
							}
							return ret
						}(),
						"annotations": map[string]interface{}{
							"getambassador.io/config": tc.inputString,
						},
					},
				},
			}

			ctx := dlog.NewTestContext(t, false)

			objs, err := snapshotTypes.ParseAnnotationResources(parentObj)
			require.NoError(t, err)
			require.Len(t, objs, 1)
			obj, err := snapshotTypes.ValidateAndConvertObject(ctx, objs[0])
			require.NoError(t, err)
			assert.Equal(t, tc.outputObj, obj)
		})
	}
}

// TestConvertAnnotationV1 checks that annotations written with the legacy `v1` apiVersion (spelled
// either `ambassador/v1` or `getambassador.io/v1`) are converted to the current version, instead of
// panicking. The `v1` Go types carry an unexported placeholder field that
// k8s.io/apimachinery/pkg/runtime's unstructured-to-typed conversion cannot set.
func TestConvertAnnotationV1(t *testing.T) {
	timeout := &amb.MillisecondDuration{Duration: 10 * time.Second}

	testcases := map[string]struct {
		inputString string
		outputObj   kates.Object
	}{
		"mapping-old-group": {
			inputString: `
---
apiVersion: ambassador/v1
kind: Mapping
name: deprecated
prefix: /deprecated
timeout_ms: 10000
service: irrelevant`,
			outputObj: &amb.Mapping{
				TypeMeta: metav1.TypeMeta{
					Kind:       "Mapping",
					APIVersion: "getambassador.io/v3alpha1",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "deprecated",
					Namespace: "somens",
					Labels:    map[string]string{},
				},
				Spec: amb.MappingSpec{
					Prefix:   "/deprecated",
					Service:  "irrelevant",
					Timeout:  timeout,
					Hostname: "*", // the v1/v2 -> v3alpha1 conversion defaults a missing host to "*"
				},
			},
		},
		"authservice": {
			inputString: `
---
apiVersion: getambassador.io/v1
kind: AuthService
name: authentication
auth_service: auth:3000
proto: grpc`,
			outputObj: &amb.AuthService{
				TypeMeta: metav1.TypeMeta{
					Kind:       "AuthService",
					APIVersion: "getambassador.io/v3alpha1",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "authentication",
					Namespace: "somens",
					Labels:    map[string]string{},
				},
				Spec: amb.AuthServiceSpec{
					AuthService: "auth:3000",
					Proto:       "grpc",
				},
			},
		},
		"module": {
			inputString: `
---
apiVersion: getambassador.io/v1
kind: Module
name: ambassador
config:
  diagnostics:
    enabled: true`,
			outputObj: &amb.Module{
				TypeMeta: metav1.TypeMeta{
					Kind:       "Module",
					APIVersion: "getambassador.io/v3alpha1",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "ambassador",
					Namespace: "somens",
					Labels:    map[string]string{},
				},
				Spec: amb.ModuleSpec{
					Config: getModuleSpec(t, `{"diagnostics":{"enabled":true}}`),
				},
			},
		},
	}

	for tcName, tc := range testcases {
		tc := tc
		t.Run(tcName, func(t *testing.T) {
			parentObj := &kates.Unstructured{
				Object: map[string]interface{}{
					"apiVersion": "v1",
					"kind":       "Service",
					"metadata": map[string]interface{}{
						"name":      "parentname",
						"namespace": "somens",
						"labels":    map[string]interface{}{},
						"annotations": map[string]interface{}{
							"getambassador.io/config": tc.inputString,
						},
					},
				},
			}

			ctx := dlog.NewTestContext(t, false)

			objs, err := snapshotTypes.ParseAnnotationResources(parentObj)
			require.NoError(t, err)
			require.Len(t, objs, 1)

			var obj kates.Object
			require.NotPanics(t, func() {
				obj, err = snapshotTypes.ValidateAndConvertObject(ctx, objs[0])
			})
			require.NoError(t, err)
			assert.Equal(t, tc.outputObj, obj)
		})
	}
}

// TestPopulateAnnotationsV1 is the end-to-end shape of the crash: a Service with a deprecated
// `ambassador/v1` annotation must not take down the whole snapshot (and with it the watcher).
func TestPopulateAnnotationsV1(t *testing.T) {
	svc := &kates.Service{
		TypeMeta: metav1.TypeMeta{
			Kind: "Service",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "svc",
			Namespace: "ambassador",
			Annotations: map[string]string{
				"getambassador.io/config": `
apiVersion: ambassador/v1
kind:  Mapping
name:  deprecated
prefix: /deprecated
timeout_ms: 10000
service: irrelevant
`,
			},
		},
	}
	ks := &snapshotTypes.KubernetesSnapshot{
		Services: []*kates.Service{svc},
	}

	ctx := dlog.NewTestContext(t, false)

	require.NotPanics(t, func() {
		assert.NoError(t, ks.PopulateAnnotations(ctx))
	})
	assert.Equal(t, map[string]snapshotTypes.AnnotationList{
		"Service/svc.ambassador": {
			&amb.Mapping{
				TypeMeta: metav1.TypeMeta{
					Kind:       "Mapping",
					APIVersion: "getambassador.io/v3alpha1",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "deprecated",
					Namespace: "ambassador",
				},
				Spec: amb.MappingSpec{
					Prefix:   "/deprecated",
					Service:  "irrelevant",
					Timeout:  &amb.MillisecondDuration{Duration: 10 * time.Second},
					Hostname: "*",
				},
			},
		},
	}, ks.Annotations)
}
