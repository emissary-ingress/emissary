package entrypoint

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/yaml"

	amb "github.com/datawire/ambassador/v2/pkg/api/getambassador.io/v2"
	"github.com/datawire/ambassador/v2/pkg/api/getambassador.io/v3alpha1"
	"github.com/datawire/ambassador/v2/pkg/kates"
	snapshotTypes "github.com/datawire/ambassador/v2/pkg/snapshot/v1"
	"github.com/datawire/dlib/dlog"
)

func getModuleSpec(rawconfig string) amb.UntypedDict {
	moduleConfig := amb.UntypedDict{}
	json.Unmarshal([]byte(rawconfig), &moduleConfig)
	return moduleConfig
}

func TestParseAnnotations(t *testing.T) {
	mapping := `
---
apiVersion: x.getambassador.io/v3alpha1
kind: AmbassadorMapping
name: quote-backend
prefix: /backend/
service: quote:80
`

	svc := &kates.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "svc",
			Namespace: "ambassador",
			Annotations: map[string]string{
				"getambassador.io/config": mapping,
			},
		},
	}

	ingHost := `
---
apiVersion: x.getambassador.io/v3alpha1
kind: AmbassadorMapping
name: cool-mapping
prefix: /blah/
`

	ingress := &kates.Ingress{
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
apiVersion: getambassador.io/v2
kind: Module
name: ambassador
config:
  diagnostics:
    enabled: true
---
apiVersion: getambassador.io/v2
kind: KubernetesEndpointResolver
name: endpoint`

	ambSvc := &kates.Service{
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
apiVersion: x.getambassador.io/v3alpha1
kind: AmbassadorMapping
name: dont-parse
prefix: /blah/`

	ignoredHost := &v3alpha1.AmbassadorHost{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "ambassador",
			Namespace: "ambassador",
			Annotations: map[string]string{
				"getambassador.io/config": unparsedAnnotation,
			},
		},
	}

	ks := &snapshotTypes.KubernetesSnapshot{
		Services:  []*kates.Service{svc, ambSvc},
		Ingresses: []*kates.Ingress{ingress},
		Hosts:     []*v3alpha1.AmbassadorHost{ignoredHost},
	}

	ctx := dlog.NewTestContext(t, false)

	parseAnnotations(ctx, ks)

	assert.NotEmpty(t, ks.Annotations)
	assert.Equal(t, len(ks.Annotations), 4)

	expectedMappings := []*v3alpha1.AmbassadorMapping{
		{
			TypeMeta: metav1.TypeMeta{
				Kind:       "AmbassadorMapping",
				APIVersion: "x.getambassador.io/v3alpha1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      "cool-mapping",
				Namespace: "somens",
			},
			Spec: v3alpha1.AmbassadorMappingSpec{
				Prefix: "/blah/",
			},
		},
		{
			TypeMeta: metav1.TypeMeta{
				Kind:       "AmbassadorMapping",
				APIVersion: "x.getambassador.io/v3alpha1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      "quote-backend",
				Namespace: "ambassador",
			},
			Spec: v3alpha1.AmbassadorMappingSpec{
				Prefix:  "/backend/",
				Service: "quote:80",
			},
		},
	}
	moduleConfigRaw := `{"diagnostics": {"enabled":true}}`
	moduleConfig := amb.UntypedDict{}
	json.Unmarshal([]byte(moduleConfigRaw), &moduleConfig)

	expectedModule := &amb.Module{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Module",
			APIVersion: "getambassador.io/v2",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "ambassador",
			Namespace: "ambassador",
		},
		Spec: amb.ModuleSpec{
			Config: getModuleSpec(`{"diagnostics":{"enabled":true}}`),
		},
	}
	expectedResolver := &amb.KubernetesEndpointResolver{
		TypeMeta: metav1.TypeMeta{
			Kind:       "KubernetesEndpointResolver",
			APIVersion: "getambassador.io/v2",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "endpoint",
			Namespace: "ambassador",
		},
	}

	foundMappings := 0
	foundModules := 0
	foundResolvers := 0
	for _, obj := range ks.Annotations {
		switch obj.(type) {
		case *v3alpha1.AmbassadorMapping:
			mapping := obj.(*v3alpha1.AmbassadorMapping)
			assert.Contains(t, expectedMappings, mapping)
			foundMappings++
		case *amb.Module:
			module := obj.(*amb.Module)
			assert.Equal(t, expectedModule, module)
			foundModules++
		case *amb.KubernetesEndpointResolver:
			res := obj.(*amb.KubernetesEndpointResolver)
			assert.Equal(t, expectedResolver, res)
			foundResolvers++
		}

	}

	assert.Equal(t, 1, foundModules)
	assert.Equal(t, 1, foundResolvers)
	assert.Equal(t, 2, foundMappings)
}

func TestConvertAnnotation(tmain *testing.T) {
	testcases := []struct {
		testName     string
		objString    string
		kind         string
		apiVersion   string
		parentns     string
		parentLabels map[string]string
		expectedObj  kates.Object
	}{
		{
			testName: "mapping",
			objString: `
---
apiVersion: x.getambassador.io/v3alpha1
kind: AmbassadorMapping
name: cool-mapping
prefix: /blah/`,
			kind:         "AmbassadorMapping",
			apiVersion:   "x.getambassador.io/v3alpha1",
			parentns:     "somens",
			parentLabels: map[string]string{},
			expectedObj: &v3alpha1.AmbassadorMapping{
				TypeMeta: metav1.TypeMeta{
					Kind:       "AmbassadorMapping",
					APIVersion: "x.getambassador.io/v3alpha1",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "cool-mapping",
					Namespace: "somens",
					Labels:    map[string]string{},
				},
				Spec: v3alpha1.AmbassadorMappingSpec{
					Prefix: "/blah/",
				},
			},
		},
		{
			testName: "old-group-v0",
			objString: `
---
apiVersion: x.getambassador.io/v3alpha1
kind: AmbassadorMapping
name: cool-mapping
prefix: /blah/`,
			kind:         "AmbassadorMapping",
			apiVersion:   "ambassador/v3alpha1",
			parentns:     "somens",
			parentLabels: map[string]string{},
			expectedObj: &v3alpha1.AmbassadorMapping{
				TypeMeta: metav1.TypeMeta{
					Kind:       "AmbassadorMapping",
					APIVersion: "x.getambassador.io/v3alpha1",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "cool-mapping",
					Namespace: "somens",
					Labels:    map[string]string{},
				},
				Spec: v3alpha1.AmbassadorMappingSpec{
					Prefix: "/blah/",
				},
			},
		},
		{
			testName: "old-group-v1",
			objString: `
---
apiVersion: x.getambassador.io/v3alpha1
kind: AmbassadorMapping
name: cool-mapping
prefix: /blah/`,
			kind:         "AmbassadorMapping",
			apiVersion:   "x.getambassador.io/v3alpha1",
			parentns:     "somens",
			parentLabels: map[string]string{},
			expectedObj: &v3alpha1.AmbassadorMapping{
				TypeMeta: metav1.TypeMeta{
					Kind:       "AmbassadorMapping",
					APIVersion: "x.getambassador.io/v3alpha1",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "cool-mapping",
					Namespace: "somens",
					Labels:    map[string]string{},
				},
				Spec: v3alpha1.AmbassadorMappingSpec{
					Prefix: "/blah/",
				},
			},
		},
		{
			testName: "label-override",
			objString: `
---
apiVersion: x.getambassador.io/v3alpha1
kind: AmbassadorMapping
name: cool-mapping
metadata_labels:
  bleep: blorp
prefix: /blah/`,
			kind:         "AmbassadorMapping",
			apiVersion:   "x.getambassador.io/v3alpha1",
			parentns:     "somens",
			parentLabels: map[string]string{"should": "override"},
			expectedObj: &v3alpha1.AmbassadorMapping{
				TypeMeta: metav1.TypeMeta{
					Kind:       "AmbassadorMapping",
					APIVersion: "x.getambassador.io/v3alpha1",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "cool-mapping",
					Namespace: "somens",
					Labels: map[string]string{
						"bleep": "blorp",
					},
				},
				Spec: v3alpha1.AmbassadorMappingSpec{
					Prefix: "/blah/",
				},
			},
		},
		{
			testName: "parent-labels",
			objString: `
---
apiVersion: x.getambassador.io/v3alpha1
kind: AmbassadorMapping
name: cool-mapping
prefix: /blah/`,
			kind:       "AmbassadorMapping",
			apiVersion: "x.getambassador.io/v3alpha1",
			parentns:   "somens",
			parentLabels: map[string]string{
				"use": "theselabels",
			},
			expectedObj: &v3alpha1.AmbassadorMapping{
				TypeMeta: metav1.TypeMeta{
					Kind:       "AmbassadorMapping",
					APIVersion: "x.getambassador.io/v3alpha1",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "cool-mapping",
					Namespace: "somens",
					Labels: map[string]string{
						"use": "theselabels",
					},
				},
				Spec: v3alpha1.AmbassadorMappingSpec{
					Prefix: "/blah/",
				},
			},
		},
		{
			testName: "module",
			objString: `
---
apiVersion: getambassador.io/v2
kind: Module
name: ambassador
config:
  diagnostics:
    enabled: true`,
			kind:         "Module",
			apiVersion:   "getambassador.io/v2",
			parentns:     "somens",
			parentLabels: map[string]string{},
			expectedObj: &amb.Module{
				TypeMeta: metav1.TypeMeta{
					Kind:       "Module",
					APIVersion: "getambassador.io/v2",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "ambassador",
					Namespace: "somens",
					Labels:    map[string]string{},
				},
				Spec: amb.ModuleSpec{
					Config: getModuleSpec(`{"diagnostics":{"enabled":true}}`),
				},
			},
		},
	}

	for _, tc := range testcases {
		tmain.Run(tc.testName, func(t *testing.T) {
			kobj := kates.NewUnstructured(tc.kind, tc.apiVersion)

			yaml.Unmarshal([]byte(tc.objString), kobj)
			parent := &kates.Service{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "svc",
					Namespace: tc.parentns,
					Labels:    tc.parentLabels,
				},
			}

			ctx := dlog.NewTestContext(t, false)

			converted := convertAnnotation(ctx, parent, kobj)

			assert.NotEmpty(t, converted)
			assert.Equal(t, tc.expectedObj, converted)
		})
	}

}
