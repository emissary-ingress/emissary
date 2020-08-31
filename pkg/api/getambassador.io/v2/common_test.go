package v2_test

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"sigs.k8s.io/yaml"

	ambV2 "github.com/datawire/ambassador/pkg/api/getambassador.io/v2"
)

func boolPtr(b bool) *bool {
	return &b
}

func stringPtr(s string) *string {
	return &s
}

func TestBoolOrString(t *testing.T) {
	t.Parallel()
	type TestResource struct {
		Field ambV2.BoolOrString `json:"field"`
	}
	type subtest struct {
		inputYAML      string
		expectedStruct TestResource
		expectedJSON   string
	}
	subtests := map[string]subtest{
		"empty":         subtest{`{}`, TestResource{}, `{"field":null}`},
		"explicitEmpty": subtest{`field:`, TestResource{}, `{"field":null}`},
		"explicitnull":  subtest{`field: null`, TestResource{}, `{"field":null}`},
		"explicitNull":  subtest{`field: Null`, TestResource{}, `{"field":null}`},
		"explicitNULL":  subtest{`field: NULL`, TestResource{}, `{"field":null}`},
		"explicitTilde": subtest{`field: ~`, TestResource{}, `{"field":null}`},
		"true":          subtest{`field: true`, TestResource{ambV2.BoolOrString{Bool: boolPtr(true)}}, `{"field":true}`},
		"True":          subtest{`field: True`, TestResource{ambV2.BoolOrString{Bool: boolPtr(true)}}, `{"field":true}`},
		"TRUE":          subtest{`field: TRUE`, TestResource{ambV2.BoolOrString{Bool: boolPtr(true)}}, `{"field":true}`},
		"false":         subtest{`field: false`, TestResource{ambV2.BoolOrString{Bool: boolPtr(false)}}, `{"field":false}`},
		"False":         subtest{`field: False`, TestResource{ambV2.BoolOrString{Bool: boolPtr(false)}}, `{"field":false}`},
		"FALSE":         subtest{`field: FALSE`, TestResource{ambV2.BoolOrString{Bool: boolPtr(false)}}, `{"field":false}`},
		"strTrue":       subtest{`field: "true"`, TestResource{ambV2.BoolOrString{String: stringPtr("true")}}, `{"field":"true"}`}, // quoted
		"strTRue":       subtest{`field: TRue`, TestResource{ambV2.BoolOrString{String: stringPtr("TRue")}}, `{"field":"TRue"}`}, // capitalized wrong
		"strBare":       subtest{`field: bare`, TestResource{ambV2.BoolOrString{String: stringPtr("bare")}}, `{"field":"bare"}`},
	}
	for name, info := range subtests {
		info := info // capture loop variable
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			var parsed TestResource
			assert.NoError(t, yaml.Unmarshal([]byte(info.inputYAML), &parsed))
			assert.Equal(t, info.expectedStruct, parsed)
			jsonbytes, err := json.Marshal(parsed)
			assert.NoError(t, err)
			assert.Equal(t, info.expectedJSON, string(jsonbytes))
		})
	}
}
