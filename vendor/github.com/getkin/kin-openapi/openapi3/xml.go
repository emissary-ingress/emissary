package openapi3

import (
	"context"

	"github.com/getkin/kin-openapi/jsoninfo"
)

// XML is specified by OpenAPI/Swagger standard version 3.
// See https://github.com/OAI/OpenAPI-Specification/blob/main/versions/3.0.3.md#xmlObject
type XML struct {
	ExtensionProps

	Name      string `json:"name,omitempty" yaml:"name,omitempty"`
	Namespace string `json:"namespace,omitempty" yaml:"namespace,omitempty"`
	Prefix    string `json:"prefix,omitempty" yaml:"prefix,omitempty"`
	Attribute bool   `json:"attribute,omitempty" yaml:"attribute,omitempty"`
	Wrapped   bool   `json:"wrapped,omitempty" yaml:"wrapped,omitempty"`
}

// MarshalJSON returns the JSON encoding of XML.
func (xml *XML) MarshalJSON() ([]byte, error) {
	return jsoninfo.MarshalStrictStruct(xml)
}

// UnmarshalJSON sets XML to a copy of data.
func (xml *XML) UnmarshalJSON(data []byte) error {
	return jsoninfo.UnmarshalStrictStruct(data, xml)
}

// Validate returns an error if XML does not comply with the OpenAPI spec.
func (xml *XML) Validate(ctx context.Context) error {
	return nil // TODO
}
