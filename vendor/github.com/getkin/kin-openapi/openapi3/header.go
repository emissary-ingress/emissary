package openapi3

import (
	"context"
	"errors"
	"fmt"

	"github.com/go-openapi/jsonpointer"

	"github.com/getkin/kin-openapi/jsoninfo"
)

type Headers map[string]*HeaderRef

var _ jsonpointer.JSONPointable = (*Headers)(nil)

// JSONLookup implements github.com/go-openapi/jsonpointer#JSONPointable
func (h Headers) JSONLookup(token string) (interface{}, error) {
	ref, ok := h[token]
	if ref == nil || !ok {
		return nil, fmt.Errorf("object has no field %q", token)
	}

	if ref.Ref != "" {
		return &Ref{Ref: ref.Ref}, nil
	}
	return ref.Value, nil
}

// Header is specified by OpenAPI/Swagger 3.0 standard.
// See https://github.com/OAI/OpenAPI-Specification/blob/main/versions/3.0.3.md#headerObject
type Header struct {
	Parameter
}

var _ jsonpointer.JSONPointable = (*Header)(nil)

// UnmarshalJSON sets Header to a copy of data.
func (header *Header) UnmarshalJSON(data []byte) error {
	return jsoninfo.UnmarshalStrictStruct(data, header)
}

// SerializationMethod returns a header's serialization method.
func (header *Header) SerializationMethod() (*SerializationMethod, error) {
	style := header.Style
	if style == "" {
		style = SerializationSimple
	}
	explode := false
	if header.Explode != nil {
		explode = *header.Explode
	}
	return &SerializationMethod{Style: style, Explode: explode}, nil
}

// Validate returns an error if Header does not comply with the OpenAPI spec.
func (header *Header) Validate(ctx context.Context) error {
	if header.Name != "" {
		return errors.New("header 'name' MUST NOT be specified, it is given in the corresponding headers map")
	}
	if header.In != "" {
		return errors.New("header 'in' MUST NOT be specified, it is implicitly in header")
	}

	// Validate a parameter's serialization method.
	sm, err := header.SerializationMethod()
	if err != nil {
		return err
	}
	if smSupported := false ||
		sm.Style == SerializationSimple && !sm.Explode ||
		sm.Style == SerializationSimple && sm.Explode; !smSupported {
		e := fmt.Errorf("serialization method with style=%q and explode=%v is not supported by a header parameter", sm.Style, sm.Explode)
		return fmt.Errorf("header schema is invalid: %v", e)
	}

	if (header.Schema == nil) == (header.Content == nil) {
		e := fmt.Errorf("parameter must contain exactly one of content and schema: %v", header)
		return fmt.Errorf("header schema is invalid: %v", e)
	}
	if schema := header.Schema; schema != nil {
		if err := schema.Validate(ctx); err != nil {
			return fmt.Errorf("header schema is invalid: %v", err)
		}
	}

	if content := header.Content; content != nil {
		if err := content.Validate(ctx); err != nil {
			return fmt.Errorf("header content is invalid: %v", err)
		}
	}
	return nil
}

// JSONLookup implements github.com/go-openapi/jsonpointer#JSONPointable
func (header Header) JSONLookup(token string) (interface{}, error) {
	switch token {
	case "schema":
		if header.Schema != nil {
			if header.Schema.Ref != "" {
				return &Ref{Ref: header.Schema.Ref}, nil
			}
			return header.Schema.Value, nil
		}
	case "name":
		return header.Name, nil
	case "in":
		return header.In, nil
	case "description":
		return header.Description, nil
	case "style":
		return header.Style, nil
	case "explode":
		return header.Explode, nil
	case "allowEmptyValue":
		return header.AllowEmptyValue, nil
	case "allowReserved":
		return header.AllowReserved, nil
	case "deprecated":
		return header.Deprecated, nil
	case "required":
		return header.Required, nil
	case "example":
		return header.Example, nil
	case "examples":
		return header.Examples, nil
	case "content":
		return header.Content, nil
	}

	v, _, err := jsonpointer.GetForToken(header.ExtensionProps, token)
	return v, err
}
