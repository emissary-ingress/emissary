package v3alpha1_test

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"sigs.k8s.io/yaml"

	ambV2 "github.com/datawire/ambassador/pkg/api/getambassador.io/v3alpha1"
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
		Field ambV2.BoolOrString `json:"field,omitempty"`
	}
	type subtest struct {
		inputYAML      string
		expectedStruct TestResource
		expectedJSON   string
	}
	subtests := map[string]subtest{
		"empty":         {`{}`, TestResource{}, `{"field":null}`},
		"explicitEmpty": {`field:`, TestResource{}, `{"field":null}`},
		"explicitnull":  {`field: null`, TestResource{}, `{"field":null}`},
		"explicitNull":  {`field: Null`, TestResource{}, `{"field":null}`},
		"explicitNULL":  {`field: NULL`, TestResource{}, `{"field":null}`},
		"explicitTilde": {`field: ~`, TestResource{}, `{"field":null}`},
		"true":          {`field: true`, TestResource{ambV2.BoolOrString{Bool: boolPtr(true)}}, `{"field":true}`},
		"True":          {`field: True`, TestResource{ambV2.BoolOrString{Bool: boolPtr(true)}}, `{"field":true}`},
		"TRUE":          {`field: TRUE`, TestResource{ambV2.BoolOrString{Bool: boolPtr(true)}}, `{"field":true}`},
		"false":         {`field: false`, TestResource{ambV2.BoolOrString{Bool: boolPtr(false)}}, `{"field":false}`},
		"False":         {`field: False`, TestResource{ambV2.BoolOrString{Bool: boolPtr(false)}}, `{"field":false}`},
		"FALSE":         {`field: FALSE`, TestResource{ambV2.BoolOrString{Bool: boolPtr(false)}}, `{"field":false}`},
		"strTrue":       {`field: "true"`, TestResource{ambV2.BoolOrString{String: stringPtr("true")}}, `{"field":"true"}`}, // quoted
		"strTRue":       {`field: TRue`, TestResource{ambV2.BoolOrString{String: stringPtr("TRue")}}, `{"field":"TRue"}`},   // capitalized wrong
		"strBare":       {`field: bare`, TestResource{ambV2.BoolOrString{String: stringPtr("bare")}}, `{"field":"bare"}`},
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

func TestBoolOrStringPtr(t *testing.T) {
	t.Parallel()
	type TestResource struct {
		Field *ambV2.BoolOrString `json:"field,omitempty"`
	}
	type subtest struct {
		inputYAML      string
		expectedStruct TestResource
		expectedJSON   string
	}
	subtests := map[string]subtest{
		"empty":         {`{}`, TestResource{}, `{}`},
		"explicitEmpty": {`field:`, TestResource{}, `{}`},
		"explicitnull":  {`field: null`, TestResource{}, `{}`},
		"explicitNull":  {`field: Null`, TestResource{}, `{}`},
		"explicitNULL":  {`field: NULL`, TestResource{}, `{}`},
		"explicitTilde": {`field: ~`, TestResource{}, `{}`},
		"true":          {`field: true`, TestResource{&ambV2.BoolOrString{Bool: boolPtr(true)}}, `{"field":true}`},
		"True":          {`field: True`, TestResource{&ambV2.BoolOrString{Bool: boolPtr(true)}}, `{"field":true}`},
		"TRUE":          {`field: TRUE`, TestResource{&ambV2.BoolOrString{Bool: boolPtr(true)}}, `{"field":true}`},
		"false":         {`field: false`, TestResource{&ambV2.BoolOrString{Bool: boolPtr(false)}}, `{"field":false}`},
		"False":         {`field: False`, TestResource{&ambV2.BoolOrString{Bool: boolPtr(false)}}, `{"field":false}`},
		"FALSE":         {`field: FALSE`, TestResource{&ambV2.BoolOrString{Bool: boolPtr(false)}}, `{"field":false}`},
		"strTrue":       {`field: "true"`, TestResource{&ambV2.BoolOrString{String: stringPtr("true")}}, `{"field":"true"}`}, // quoted
		"strTRue":       {`field: TRue`, TestResource{&ambV2.BoolOrString{String: stringPtr("TRue")}}, `{"field":"TRue"}`},   // capitalized wrong
		"strBare":       {`field: bare`, TestResource{&ambV2.BoolOrString{String: stringPtr("bare")}}, `{"field":"bare"}`},
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
