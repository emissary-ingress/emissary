package kates

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"k8s.io/api/core/v1"
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
