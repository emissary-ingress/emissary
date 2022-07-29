package openapi3

import (
	"context"
	"fmt"

	"github.com/getkin/kin-openapi/jsoninfo"
)

// Tags is specified by OpenAPI/Swagger 3.0 standard.
type Tags []*Tag

func (tags Tags) Get(name string) *Tag {
	for _, tag := range tags {
		if tag.Name == name {
			return tag
		}
	}
	return nil
}

// Validate returns an error if Tags does not comply with the OpenAPI spec.
func (tags Tags) Validate(ctx context.Context) error {
	for _, v := range tags {
		if err := v.Validate(ctx); err != nil {
			return err
		}
	}
	return nil
}

// Tag is specified by OpenAPI/Swagger 3.0 standard.
// See https://github.com/OAI/OpenAPI-Specification/blob/main/versions/3.0.3.md#tagObject
type Tag struct {
	ExtensionProps

	Name         string        `json:"name,omitempty" yaml:"name,omitempty"`
	Description  string        `json:"description,omitempty" yaml:"description,omitempty"`
	ExternalDocs *ExternalDocs `json:"externalDocs,omitempty" yaml:"externalDocs,omitempty"`
}

// MarshalJSON returns the JSON encoding of Tag.
func (t *Tag) MarshalJSON() ([]byte, error) {
	return jsoninfo.MarshalStrictStruct(t)
}

// UnmarshalJSON sets Tag to a copy of data.
func (t *Tag) UnmarshalJSON(data []byte) error {
	return jsoninfo.UnmarshalStrictStruct(data, t)
}

// Validate returns an error if Tag does not comply with the OpenAPI spec.
func (t *Tag) Validate(ctx context.Context) error {
	if v := t.ExternalDocs; v != nil {
		if err := v.Validate(ctx); err != nil {
			return fmt.Errorf("invalid external docs: %w", err)
		}
	}
	return nil
}
