package kates

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"k8s.io/api/core/v1"
)

var svc = `
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
