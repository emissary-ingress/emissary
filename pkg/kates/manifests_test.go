package kates

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	gw "sigs.k8s.io/gateway-api/apis/v1alpha1"

	amb "github.com/emissary-ingress/emissary/v3/pkg/api/getambassador.io/v3alpha1"
)

func TestMergeUpdate(t *testing.T) {
	a := &Unstructured{}
	a.Object = map[string]interface{}{
		"apiVersion": "vtest",
		"kind":       "Foo",
		"metadata": map[string]interface{}{
			"name":            "foo",
			"namespace":       "default",
			"resourceVersion": "1234",
			"labels": map[string]interface{}{
				"foo": "bar",
			},
			"annotations": map[string]interface{}{
				"moo": "arf",
			},
		},
		"spec": map[string]interface{}{
			"foo": "bar",
		},
	}

	b := &Unstructured{}
	b.Object = map[string]interface{}{
		"apiVersion": "vtest",
		"kind":       "Foo",
		"metadata": map[string]interface{}{
			"labels": map[string]interface{}{
				"foo": "bar",
				"moo": "arf",
			},
			"annotations": map[string]interface{}{
				"foo": "bar",
				"moo": "arf",
			},
		},
		"spec": map[string]interface{}{
			"key": "value",
		},
	}

	assert.NotEqual(t, a.Object["spec"], b.Object["spec"])

	MergeUpdate(a, b)

	assert.Equal(t, "arf", a.GetLabels()["moo"])
	assert.Equal(t, "bar", a.GetAnnotations()["foo"])
	assert.Equal(t, b.Object["spec"], a.Object["spec"])
}

const mapping = `---
apiVersion: getambassador.io/v3alpha1
kind: Mapping
metadata:
  name:  mapping-name
spec:
  prefix: /mapping-prefix/
  service: http://mapping-service
`

func TestParseManifestsResultTypes(t *testing.T) {
	objs, err := ParseManifests(mapping)
	require.NoError(t, err)
	require.Equal(t, 1, len(objs))

	m := objs[0]
	t.Logf("value = %v", m)
	t.Logf("type = %T", m)
	_, ok := m.(*amb.Mapping)
	require.True(t, ok)
}

// TestGatewayResources checks that the kates schema includes the gateway resources and will
// therefore parse into the proper types as opposed to falling back to the Unstructured type.
func TestGatewayResources(t *testing.T) {
	objs, err := ParseManifests(gatewayResources)
	require.NoError(t, err)
	require.Len(t, objs, 3)
	assert.IsType(t, &gw.GatewayClass{}, objs[0])
	assert.IsType(t, &gw.Gateway{}, objs[1])
	assert.IsType(t, &gw.HTTPRoute{}, objs[2])
}

const gatewayResources = `
---
kind: GatewayClass
apiVersion: networking.x-k8s.io/v1alpha1
metadata:
  name: acme-lb
spec:
  controller: acme.io/gateway-controller
  parametersRef:
    name: acme-lb
    group: acme.io
    kind: Parameters
---
kind: Gateway
apiVersion: networking.x-k8s.io/v1alpha1
metadata:
  name: my-gateway
spec:
  gatewayClassName: acme-lb
  listeners:  # Use GatewayClass defaults for listener definition.
  - protocol: HTTP
    port: 80
    routes:
      kind: HTTPRoute
      selector:
        matchLabels:
          app: foo
      namespaces:
        from: "Same"
---
kind: HTTPRoute
apiVersion: networking.x-k8s.io/v1alpha1
metadata:
  name: http-app-1
  labels:
    app: foo
spec:
  hostnames:
  - "foo.com"
  rules:
  - matches:
    - path:
        type: Prefix
        value: /bar
    forwardTo:
    - serviceName: my-service1
      port: 8080
  - matches:
    - headers:
        type: Exact
        values:
          magic: foo
      path:
        type: Prefix
        value: /some/thing
    forwardTo:
    - serviceName: my-service2
      port: 8080
`
