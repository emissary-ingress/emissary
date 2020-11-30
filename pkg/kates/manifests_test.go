package kates

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/api/core/v1"

	amb "github.com/datawire/ambassador/pkg/api/getambassador.io/v2"
)

var svc = `
# leading comment
---
apiVersion: v1
kind: Service
metadata:
  name: example-service
  namespace: foo
`

func TestByName(t *testing.T) {
	objs, err := ParseManifests(svc)
	if err != nil {
		panic(err)
	}
	_ = objs[0].(*v1.Service)
	svcs := make(map[string]*Service)
	ByName(objs, svcs)
	assert.Equal(t, 1, len(svcs))
	assert.Equal(t, objs[0], svcs["example-service"])
}

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
apiVersion: getambassador.io/v2
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
	t.Log(m)
	t.Log(reflect.TypeOf(m))
	_, ok := m.(*amb.Mapping)
	require.True(t, ok)
}
