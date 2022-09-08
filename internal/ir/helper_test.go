package ir_test

import (
	"testing"

	"github.com/emissary-ingress/emissary/v3/internal/ir"
	"github.com/emissary-ingress/emissary/v3/internal/ir/types"
	"github.com/stretchr/testify/assert"
)

func TestMapToNamespacedName(t *testing.T) {
	type testCase struct {
		Description string
		Input       map[string]interface{}
		Expected    types.NamespacedName
	}

	testCases := []testCase{
		{
			Description: "name-with-namespace",
			Input:       map[string]interface{}{"name": "goku", "namespace": "dragonball"},
			Expected:    types.NamespacedName{Name: "goku", Namespace: "dragonball"},
		},
		{
			Description: "name-only",
			Input:       map[string]interface{}{"name": "goku"},
			Expected:    types.NamespacedName{Name: "goku", Namespace: ""},
		},
		{
			Description: "namespace-only",
			Input:       map[string]interface{}{"namespace": "dragonball"},
			Expected:    types.NamespacedName{Name: "", Namespace: "dragonball"},
		},
		{
			Description: "empty-map",
			Input:       make(map[string]interface{}),
			Expected:    types.NamespacedName{Name: "", Namespace: ""},
		},
		{
			Description: "no-matching-fields",
			Input:       map[string]interface{}{"fieldA": "roshi", "fieldB": "dragonball"},
			Expected:    types.NamespacedName{Name: "", Namespace: ""},
		},
		{
			Description: "invalid-field-types",
			Input:       map[string]interface{}{"name": struct{}{}, "namespace": struct{}{}},
			Expected:    types.NamespacedName{Name: "", Namespace: ""},
		},
		{
			Description: "field-nil",
			Input:       map[string]interface{}{"name": nil, "namespace": nil},
			Expected:    types.NamespacedName{Name: "", Namespace: ""},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.Description, func(tt *testing.T) {
			actual := ir.MapToNamespacedName(tc.Input)
			assert.Equal(t, tc.Expected, actual)
		})
	}
}
