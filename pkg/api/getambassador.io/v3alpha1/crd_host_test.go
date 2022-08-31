package v3alpha1_test

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"sigs.k8s.io/yaml"

	crds "github.com/emissary-ingress/emissary/v3/pkg/api/getambassador.io/v3alpha1"
)

func TestHostState(t *testing.T) {
	t.Parallel()
	type TestResource struct {
		Field crds.HostState `json:"field,omitempty"`
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

		"Initial": {`field: "Initial"`, TestResource{crds.HostState_Initial}, `{}`, ``},
		"Pending": {`field: "Pending"`, TestResource{crds.HostState_Pending}, `{"field":"Pending"}`, ``},
		"Ready":   {`field: "Ready"`, TestResource{crds.HostState_Ready}, `{"field":"Ready"}`, ``},
		"Error":   {`field: "Error"`, TestResource{crds.HostState_Error}, `{"field":"Error"}`, ``},

		"invalid-string": {`field: "invalid"`, TestResource{crds.HostState_Initial}, `{}`, ``}, // no error
		"invalid-type":   {`field: {}`, TestResource{crds.HostState_Initial}, `{}`, `error unmarshaling JSON: while decoding JSON: json: cannot unmarshal object into Go struct field TestResource.field of type string`},
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

func TestHostPhase(t *testing.T) {
	t.Parallel()
	type TestResource struct {
		Field crds.HostPhase `json:"field,omitempty"`
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

		"NA":                        {`field: "NA"`, TestResource{crds.HostPhase_NA}, `{}`, ``},
		"DefaultsFilled":            {`field: "DefaultsFilled"`, TestResource{crds.HostPhase_DefaultsFilled}, `{"field":"DefaultsFilled"}`, ``},
		"ACMEUserPrivateKeyCreated": {`field: "ACMEUserPrivateKeyCreated"`, TestResource{crds.HostPhase_ACMEUserPrivateKeyCreated}, `{"field":"ACMEUserPrivateKeyCreated"}`, ``},
		"ACMEUserRegistered":        {`field: "ACMEUserRegistered"`, TestResource{crds.HostPhase_ACMEUserRegistered}, `{"field":"ACMEUserRegistered"}`, ``},
		"ACMECertificateChallenge":  {`field: "ACMECertificateChallenge"`, TestResource{crds.HostPhase_ACMECertificateChallenge}, `{"field":"ACMECertificateChallenge"}`, ``},

		"invalid-string": {`field: "invalid"`, TestResource{crds.HostPhase_NA}, `{}`, ``}, // no error
		"invalid-type":   {`field: {}`, TestResource{crds.HostPhase_NA}, `{}`, `error unmarshaling JSON: while decoding JSON: json: cannot unmarshal object into Go struct field TestResource.field of type string`},
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
