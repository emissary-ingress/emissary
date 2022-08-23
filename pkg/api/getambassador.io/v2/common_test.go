package v2_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"sigs.k8s.io/yaml"

	crds "github.com/datawire/ambassador/v2/pkg/api/getambassador.io/v2"
)

func boolPtr(b bool) *bool {
	return &b
}

func stringPtr(s string) *string {
	return &s
}

func TestStringOrStringList(t *testing.T) {
	t.Parallel()
	type TestResource struct {
		Field crds.StringOrStringList `json:"field,omitempty"`
	}
	type subtest struct {
		inputYAML      string
		expectedStruct TestResource
		expectedJSON   string
		expectedErr    string
	}
	subtests := map[string]subtest{
		"empty":         {`{}`, TestResource{}, `{}`, ``},
		"explicitEmpty": {`field:`, TestResource{}, `{}`, ``},
		"explicitnull":  {`field: null`, TestResource{}, `{}`, ``},
		"explicitNull":  {`field: Null`, TestResource{}, `{}`, ``},
		"explicitNULL":  {`field: NULL`, TestResource{}, `{}`, ``},
		"explicitTilde": {`field: ~`, TestResource{}, `{}`, ``},
		"single":        {`field: "single"`, TestResource{crds.StringOrStringList{"single"}}, `{"field":["single"]}`, ``},
		"singleEmpty":   {`field: ""`, TestResource{crds.StringOrStringList{""}}, `{"field":[""]}`, ``},
		"listSingle":    {`field: ["single"]`, TestResource{crds.StringOrStringList{"single"}}, `{"field":["single"]}`, ``},
		"listEmpty":     {`field: []`, TestResource{crds.StringOrStringList{}}, `{}`, ``},
		"double":        {`field: ["first", "second"]`, TestResource{crds.StringOrStringList{"first", "second"}}, `{"field":["first","second"]}`, ``},
		"number":        {`field: 12`, TestResource{}, `{}`, "error unmarshaling JSON: while decoding JSON: json: cannot unmarshal number into Go struct field TestResource.field of type []string"},
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

func TestBoolOrString(t *testing.T) {
	t.Parallel()
	type TestResource struct {
		Field crds.BoolOrString `json:"field,omitempty"`
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
		"true":          {`field: true`, TestResource{crds.BoolOrString{Bool: boolPtr(true)}}, `{"field":true}`, ``},
		"True":          {`field: True`, TestResource{crds.BoolOrString{Bool: boolPtr(true)}}, `{"field":true}`, ``},
		"TRUE":          {`field: TRUE`, TestResource{crds.BoolOrString{Bool: boolPtr(true)}}, `{"field":true}`, ``},
		"false":         {`field: false`, TestResource{crds.BoolOrString{Bool: boolPtr(false)}}, `{"field":false}`, ``},
		"False":         {`field: False`, TestResource{crds.BoolOrString{Bool: boolPtr(false)}}, `{"field":false}`, ``},
		"FALSE":         {`field: FALSE`, TestResource{crds.BoolOrString{Bool: boolPtr(false)}}, `{"field":false}`, ``},
		"strTrue":       {`field: "true"`, TestResource{crds.BoolOrString{String: stringPtr("true")}}, `{"field":"true"}`, ``}, // quoted
		"strTRue":       {`field: TRue`, TestResource{crds.BoolOrString{String: stringPtr("TRue")}}, `{"field":"TRue"}`, ``},   // capitalized wrong
		"strBare":       {`field: bare`, TestResource{crds.BoolOrString{String: stringPtr("bare")}}, `{"field":"bare"}`, ``},
		"number":        {`field: 12`, TestResource{}, `{"field":null}`, "error unmarshaling JSON: while decoding JSON: json: cannot unmarshal number into Go struct field TestResource.field of type string"},
		"invalid":       {``, TestResource{crds.BoolOrString{Bool: boolPtr(true), String: stringPtr("foo")}}, ``, "json: error calling MarshalJSON for type v2.BoolOrString: invalid BoolOrString"},
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

func TestBoolOrStringPtr(t *testing.T) {
	t.Parallel()
	type TestResource struct {
		Field *crds.BoolOrString `json:"field,omitempty"`
	}
	type subtest struct {
		inputYAML      string
		expectedStruct TestResource
		expectedJSON   string
		expectedErr    string
	}
	subtests := map[string]subtest{
		"empty":         {`{}`, TestResource{}, `{}`, ``},
		"explicitEmpty": {`field:`, TestResource{}, `{}`, ``},
		"explicitnull":  {`field: null`, TestResource{}, `{}`, ``},
		"explicitNull":  {`field: Null`, TestResource{}, `{}`, ``},
		"explicitNULL":  {`field: NULL`, TestResource{}, `{}`, ``},
		"explicitTilde": {`field: ~`, TestResource{}, `{}`, ``},
		"true":          {`field: true`, TestResource{&crds.BoolOrString{Bool: boolPtr(true)}}, `{"field":true}`, ``},
		"True":          {`field: True`, TestResource{&crds.BoolOrString{Bool: boolPtr(true)}}, `{"field":true}`, ``},
		"TRUE":          {`field: TRUE`, TestResource{&crds.BoolOrString{Bool: boolPtr(true)}}, `{"field":true}`, ``},
		"false":         {`field: false`, TestResource{&crds.BoolOrString{Bool: boolPtr(false)}}, `{"field":false}`, ``},
		"False":         {`field: False`, TestResource{&crds.BoolOrString{Bool: boolPtr(false)}}, `{"field":false}`, ``},
		"FALSE":         {`field: FALSE`, TestResource{&crds.BoolOrString{Bool: boolPtr(false)}}, `{"field":false}`, ``},
		"strTrue":       {`field: "true"`, TestResource{&crds.BoolOrString{String: stringPtr("true")}}, `{"field":"true"}`, ``}, // quoted
		"strTRue":       {`field: TRue`, TestResource{&crds.BoolOrString{String: stringPtr("TRue")}}, `{"field":"TRue"}`, ``},   // capitalized wrong
		"strBare":       {`field: bare`, TestResource{&crds.BoolOrString{String: stringPtr("bare")}}, `{"field":"bare"}`, ``},
		"number":        {`field: 12`, TestResource{&crds.BoolOrString{}}, `{"field":null}`, "error unmarshaling JSON: while decoding JSON: json: cannot unmarshal number into Go struct field TestResource.field of type string"},
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

func TestMillisecondDuration(t *testing.T) {
	t.Parallel()
	type TestResource struct {
		Field crds.MillisecondDuration `json:"field,omitempty"`
	}
	type subtest struct {
		inputYAML      string
		expectedStruct TestResource
		expectedJSON   string
		expectedErr    string
	}
	subtests := map[string]subtest{
		"empty":         {`{}`, TestResource{}, `{"field":0}`, ``},
		"explicitEmpty": {`field:`, TestResource{}, `{"field":0}`, ``},
		"explicitnull":  {`field: null`, TestResource{}, `{"field":0}`, ``},
		"explicitNull":  {`field: Null`, TestResource{}, `{"field":0}`, ``},
		"explicitNULL":  {`field: NULL`, TestResource{}, `{"field":0}`, ``},
		"explicitTilde": {`field: ~`, TestResource{}, `{"field":0}`, ``},
		"3000":          {`field: 3000`, TestResource{crds.MillisecondDuration{3 * time.Second}}, `{"field":3000}`, ``},
		"overflow32":    {`field: 4320000000`, TestResource{crds.MillisecondDuration{50 * 24 * time.Hour}}, `{"field":4320000000}`, ``},
		"string":        {`field: "30s"`, TestResource{crds.MillisecondDuration{0}}, `{"field":0}`, "error unmarshaling JSON: while decoding JSON: json: cannot unmarshal string into Go struct field TestResource.field of type int64"},
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

func TestMillisecondDurationPtr(t *testing.T) {
	t.Parallel()
	type TestResource struct {
		Field *crds.MillisecondDuration `json:"field,omitempty"`
	}
	type subtest struct {
		inputYAML      string
		expectedStruct TestResource
		expectedJSON   string
		expectedErr    string
	}
	subtests := map[string]subtest{
		"empty":         {`{}`, TestResource{}, `{}`, ``},
		"explicitEmpty": {`field:`, TestResource{}, `{}`, ``},
		"explicitnull":  {`field: null`, TestResource{}, `{}`, ``},
		"explicitNull":  {`field: Null`, TestResource{}, `{}`, ``},
		"explicitNULL":  {`field: NULL`, TestResource{}, `{}`, ``},
		"explicitTilde": {`field: ~`, TestResource{}, `{}`, ``},
		"3000":          {`field: 3000`, TestResource{&crds.MillisecondDuration{3 * time.Second}}, `{"field":3000}`, ``},
		"overflow32":    {`field: 4320000000`, TestResource{&crds.MillisecondDuration{50 * 24 * time.Hour}}, `{"field":4320000000}`, ``},
		"string":        {`field: "30s"`, TestResource{&crds.MillisecondDuration{0}}, `{"field":0}`, "error unmarshaling JSON: while decoding JSON: json: cannot unmarshal string into Go struct field TestResource.field of type int64"},
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

func TestUntypedDict(t *testing.T) {
	t.Parallel()
	type TestResource struct {
		Field crds.UntypedDict `json:"field,omitempty"`
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
		"basic":         {`field: {foo: "bar"}`, TestResource{crds.UntypedDict{map[string]json.RawMessage{"foo": json.RawMessage(`"bar"`)}}}, `{"field":{"foo":"bar"}}`, ``},
		"string":        {`field: "str"`, TestResource{}, `{"field":null}`, "error unmarshaling JSON: while decoding JSON: json: cannot unmarshal string into Go struct field TestResource.field of type map[string]json.RawMessage"},
		"badkey": {`
badkey: &anchor
  baz: qux
field: { *anchor: "bar"}`, TestResource{}, `{"field":null}`, "error converting YAML to JSON: yaml: invalid map key: map[interface {}]interface {}{\"baz\":\"qux\"}"},
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

func TestUntypedDictPtr(t *testing.T) {
	t.Parallel()
	type TestResource struct {
		Field *crds.UntypedDict `json:"field,omitempty"`
	}
	type subtest struct {
		inputYAML      string
		expectedStruct TestResource
		expectedJSON   string
		expectedErr    string
	}
	subtests := map[string]subtest{
		"empty":         {`{}`, TestResource{}, `{}`, ``},
		"explicitEmpty": {`field:`, TestResource{}, `{}`, ``},
		"explicitnull":  {`field: null`, TestResource{}, `{}`, ``},
		"explicitNull":  {`field: Null`, TestResource{}, `{}`, ``},
		"explicitNULL":  {`field: NULL`, TestResource{}, `{}`, ``},
		"explicitTilde": {`field: ~`, TestResource{}, `{}`, ``},
		"basic":         {`field: {foo: "bar"}`, TestResource{&crds.UntypedDict{map[string]json.RawMessage{"foo": json.RawMessage(`"bar"`)}}}, `{"field":{"foo":"bar"}}`, ``},
		"string":        {`field: "str"`, TestResource{&crds.UntypedDict{}}, `{"field":null}`, "error unmarshaling JSON: while decoding JSON: json: cannot unmarshal string into Go struct field TestResource.field of type map[string]json.RawMessage"},
		"badkey": {`
badkey: &anchor
  baz: qux
field: { *anchor: "bar"}`, TestResource{}, `{}`, "error converting YAML to JSON: yaml: invalid map key: map[interface {}]interface {}{\"baz\":\"qux\"}"},
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
