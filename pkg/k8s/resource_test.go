package k8s_test

import (
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/datawire/ambassador/v2/pkg/k8s"
)

func TestQKind(t *testing.T) {
	t.Parallel()
	testcases := []struct {
		InputResource k8s.Resource
		ExpectedQKind string
	}{
		// Sane things we need to handle correcly
		{k8s.Resource{"apiVersion": "apps/v1", "kind": "Deployment"}, "Deployment.v1.apps"},
		{k8s.Resource{"apiVersion": "v1", "kind": "Service"}, "Service.v1."},
		// Insane things that shouldn't happen, but at least our function is well-defined
		{k8s.Resource{"kind": "KindOnly"}, "KindOnly.."},
		{k8s.Resource{"apiVersion": "group/version"}, ".version.group"},
		{k8s.Resource{}, ".."},
		{k8s.Resource{"kind": 7, "apiVersion": "v1"}, ".v1."},
		{k8s.Resource{"kind": "Pod", "apiVersion": 1}, "Pod.."},
	}
	for i, testcase := range testcases {
		testcase := testcase
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			actualQKind := testcase.InputResource.QKind()
			assert.Equal(t, testcase.ExpectedQKind, actualQKind)
		})
	}
}
