package v2_test

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"sigs.k8s.io/yaml"

	crds "github.com/emissary-ingress/emissary/v3/pkg/api/getambassador.io/v2"
)

func TestMappingLabelSpecifier(t *testing.T) {
	t.Parallel()
	type TestResource struct {
		Field crds.MappingLabelSpecifier `json:"field,omitempty"`
	}
	type subtest struct {
		inputYAML      string
		expectedStruct TestResource
		expectedJSON   string
		expectedErr    string
	}
	subtests := map[string]subtest{
		"empty":         {`{}`, TestResource{}, `{"field":null}`, ``},
		"explicitEmpty": {`field:`, TestResource{}, `{"field":null}`, ``},
		"explicitnull":  {`field: null`, TestResource{}, `{"field":null}`, ``},
		"explicitNull":  {`field: Null`, TestResource{}, `{"field":null}`, ``},
		"explicitNULL":  {`field: NULL`, TestResource{}, `{"field":null}`, ``},
		"explicitTilde": {`field: ~`, TestResource{}, `{"field":null}`, ``},
		"list": {`field: ["foo"]`, TestResource{}, `{"field":null}`,
			"error unmarshaling JSON: while decoding JSON: could not unmarshal MappingLabelSpecifier: invalid input"},
		"invalid": {``, TestResource{crds.MappingLabelSpecifier{String: stringPtr("foo"), Generic: &crds.MappingLabelSpecGeneric{GenericKey: "foo"}}}, ``,
			"json: error calling MarshalJSON for type v2.MappingLabelSpecifier: invalid MappingLabelSpecifier"},
	}
	for name, info := range subtests {
		info := info // capture loop variable
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			if info.inputYAML == `` {
				jsonbytes, err := json.Marshal(info.expectedStruct)
				assert.Equal(t, info.expectedJSON, string(jsonbytes))
				assert.EqualError(t, err, info.expectedErr)
			} else {
				var parsed TestResource
				err := yaml.Unmarshal([]byte(info.inputYAML), &parsed)
				if info.expectedErr == `` {
					assert.NoError(t, err)
				} else {
					assert.EqualError(t, err, info.expectedErr)
				}
				assert.Equal(t, info.expectedStruct, parsed)
				jsonbytes, err := json.Marshal(parsed)
				assert.NoError(t, err)
				assert.Equal(t, info.expectedJSON, string(jsonbytes))
			}
		})
	}
}

func TestAddedHeader(t *testing.T) {
	t.Parallel()
	var testResource struct {
		Field crds.AddedHeader `json:"field,omitempty"`
	}
	assert.EqualError(t, yaml.Unmarshal([]byte(`field: ["foo"]`), &testResource),
		"error unmarshaling JSON: while decoding JSON: json: cannot unmarshal array into Go struct field .field of type v2.AddedHeaderFull")
}

func TestOriginList(t *testing.T) {
	t.Parallel()
	type TestResource struct {
		Field crds.OriginList `json:"field,omitempty"`
	}
	type subtest struct {
		inputYAML      string
		expectedStruct TestResource
		expectedJSON   string
		expectedErr    string
	}
	subtests := map[string]subtest{
		"empty":         {`{}`, TestResource{}, `{"field":null}`, ``},
		"explicitEmpty": {`field:`, TestResource{}, `{"field":null}`, ``},
		"explicitnull":  {`field: null`, TestResource{}, `{"field":null}`, ``},
		"explicitNull":  {`field: Null`, TestResource{}, `{"field":null}`, ``},
		"explicitNULL":  {`field: NULL`, TestResource{}, `{"field":null}`, ``},
		"explicitTilde": {`field: ~`, TestResource{}, `{"field":null}`, ``},
		"object": {`field: {"foo": "bar"}`, TestResource{}, `{"field":null}`,
			"error unmarshaling JSON: while decoding JSON: json: cannot unmarshal object into Go struct field TestResource.field of type []string"},
	}
	for name, info := range subtests {
		info := info // capture loop variable
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			var parsed TestResource
			err := yaml.Unmarshal([]byte(info.inputYAML), &parsed)
			if info.expectedErr == `` {
				assert.NoError(t, err)
			} else {
				assert.EqualError(t, err, info.expectedErr)
			}
			assert.Equal(t, info.expectedStruct, parsed)
			jsonbytes, err := json.Marshal(parsed)
			assert.NoError(t, err)
			assert.Equal(t, info.expectedJSON, string(jsonbytes))
		})
	}
}
