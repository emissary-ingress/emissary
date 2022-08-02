package openapi3

import (
	"context"
	"fmt"

	"github.com/go-openapi/jsonpointer"

	"github.com/getkin/kin-openapi/jsoninfo"
)

type Examples map[string]*ExampleRef

var _ jsonpointer.JSONPointable = (*Examples)(nil)

// JSONLookup implements github.com/go-openapi/jsonpointer#JSONPointable
func (e Examples) JSONLookup(token string) (interface{}, error) {
	ref, ok := e[token]
	if ref == nil || !ok {
		return nil, fmt.Errorf("object has no field %q", token)
	}

	if ref.Ref != "" {
		return &Ref{Ref: ref.Ref}, nil
	}
	return ref.Value, nil
}

// Example is specified by OpenAPI/Swagger 3.0 standard.
// See https://github.com/OAI/OpenAPI-Specification/blob/main/versions/3.0.3.md#exampleObject
type Example struct {
	ExtensionProps

	Summary       string      `json:"summary,omitempty" yaml:"summary,omitempty"`
	Description   string      `json:"description,omitempty" yaml:"description,omitempty"`
	Value         interface{} `json:"value,omitempty" yaml:"value,omitempty"`
	ExternalValue string      `json:"externalValue,omitempty" yaml:"externalValue,omitempty"`
}

func NewExample(value interface{}) *Example {
	return &Example{
		Value: value,
	}
}

// MarshalJSON returns the JSON encoding of Example.
func (example *Example) MarshalJSON() ([]byte, error) {
	return jsoninfo.MarshalStrictStruct(example)
}

// UnmarshalJSON sets Example to a copy of data.
func (example *Example) UnmarshalJSON(data []byte) error {
	return jsoninfo.UnmarshalStrictStruct(data, example)
}

// Validate returns an error if Example does not comply with the OpenAPI spec.
func (example *Example) Validate(ctx context.Context) error {
	return nil // TODO
}
