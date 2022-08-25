package openapi3

import (
	"context"

	"github.com/go-openapi/jsonpointer"

	"github.com/getkin/kin-openapi/jsoninfo"
)

// Ref is specified by OpenAPI/Swagger 3.0 standard.
// See https://github.com/OAI/OpenAPI-Specification/blob/main/versions/3.0.3.md#referenceObject
type Ref struct {
	Ref string `json:"$ref" yaml:"$ref"`
}

// CallbackRef represents either a Callback or a $ref to a Callback.
// When serializing and both fields are set, Ref is preferred over Value.
type CallbackRef struct {
	Ref   string
	Value *Callback
}

var _ jsonpointer.JSONPointable = (*CallbackRef)(nil)

// MarshalJSON returns the JSON encoding of CallbackRef.
func (value *CallbackRef) MarshalJSON() ([]byte, error) {
	return jsoninfo.MarshalRef(value.Ref, value.Value)
}

// UnmarshalJSON sets CallbackRef to a copy of data.
func (value *CallbackRef) UnmarshalJSON(data []byte) error {
	return jsoninfo.UnmarshalRef(data, &value.Ref, &value.Value)
}

// Validate returns an error if CallbackRef does not comply with the OpenAPI spec.
func (value *CallbackRef) Validate(ctx context.Context) error {
	if v := value.Value; v != nil {
		return v.Validate(ctx)
	}
	return foundUnresolvedRef(value.Ref)
}

// JSONLookup implements github.com/go-openapi/jsonpointer#JSONPointable
func (value CallbackRef) JSONLookup(token string) (interface{}, error) {
	if token == "$ref" {
		return value.Ref, nil
	}

	ptr, _, err := jsonpointer.GetForToken(value.Value, token)
	return ptr, err
}

// ExampleRef represents either a Example or a $ref to a Example.
// When serializing and both fields are set, Ref is preferred over Value.
type ExampleRef struct {
	Ref   string
	Value *Example
}

var _ jsonpointer.JSONPointable = (*ExampleRef)(nil)

// MarshalJSON returns the JSON encoding of ExampleRef.
func (value *ExampleRef) MarshalJSON() ([]byte, error) {
	return jsoninfo.MarshalRef(value.Ref, value.Value)
}

// UnmarshalJSON sets ExampleRef to a copy of data.
func (value *ExampleRef) UnmarshalJSON(data []byte) error {
	return jsoninfo.UnmarshalRef(data, &value.Ref, &value.Value)
}

// Validate returns an error if ExampleRef does not comply with the OpenAPI spec.
func (value *ExampleRef) Validate(ctx context.Context) error {
	if v := value.Value; v != nil {
		return v.Validate(ctx)
	}
	return foundUnresolvedRef(value.Ref)
}

// JSONLookup implements github.com/go-openapi/jsonpointer#JSONPointable
func (value ExampleRef) JSONLookup(token string) (interface{}, error) {
	if token == "$ref" {
		return value.Ref, nil
	}

	ptr, _, err := jsonpointer.GetForToken(value.Value, token)
	return ptr, err
}

// HeaderRef represents either a Header or a $ref to a Header.
// When serializing and both fields are set, Ref is preferred over Value.
type HeaderRef struct {
	Ref   string
	Value *Header
}

var _ jsonpointer.JSONPointable = (*HeaderRef)(nil)

// MarshalJSON returns the JSON encoding of HeaderRef.
func (value *HeaderRef) MarshalJSON() ([]byte, error) {
	return jsoninfo.MarshalRef(value.Ref, value.Value)
}

// UnmarshalJSON sets HeaderRef to a copy of data.
func (value *HeaderRef) UnmarshalJSON(data []byte) error {
	return jsoninfo.UnmarshalRef(data, &value.Ref, &value.Value)
}

// Validate returns an error if HeaderRef does not comply with the OpenAPI spec.
func (value *HeaderRef) Validate(ctx context.Context) error {
	if v := value.Value; v != nil {
		return v.Validate(ctx)
	}
	return foundUnresolvedRef(value.Ref)
}

// JSONLookup implements github.com/go-openapi/jsonpointer#JSONPointable
func (value HeaderRef) JSONLookup(token string) (interface{}, error) {
	if token == "$ref" {
		return value.Ref, nil
	}

	ptr, _, err := jsonpointer.GetForToken(value.Value, token)
	return ptr, err
}

// LinkRef represents either a Link or a $ref to a Link.
// When serializing and both fields are set, Ref is preferred over Value.
type LinkRef struct {
	Ref   string
	Value *Link
}

// MarshalJSON returns the JSON encoding of LinkRef.
func (value *LinkRef) MarshalJSON() ([]byte, error) {
	return jsoninfo.MarshalRef(value.Ref, value.Value)
}

// UnmarshalJSON sets LinkRef to a copy of data.
func (value *LinkRef) UnmarshalJSON(data []byte) error {
	return jsoninfo.UnmarshalRef(data, &value.Ref, &value.Value)
}

// Validate returns an error if LinkRef does not comply with the OpenAPI spec.
func (value *LinkRef) Validate(ctx context.Context) error {
	if v := value.Value; v != nil {
		return v.Validate(ctx)
	}
	return foundUnresolvedRef(value.Ref)
}

// ParameterRef represents either a Parameter or a $ref to a Parameter.
// When serializing and both fields are set, Ref is preferred over Value.
type ParameterRef struct {
	Ref   string
	Value *Parameter
}

var _ jsonpointer.JSONPointable = (*ParameterRef)(nil)

// MarshalJSON returns the JSON encoding of ParameterRef.
func (value *ParameterRef) MarshalJSON() ([]byte, error) {
	return jsoninfo.MarshalRef(value.Ref, value.Value)
}

// UnmarshalJSON sets ParameterRef to a copy of data.
func (value *ParameterRef) UnmarshalJSON(data []byte) error {
	return jsoninfo.UnmarshalRef(data, &value.Ref, &value.Value)
}

// Validate returns an error if ParameterRef does not comply with the OpenAPI spec.
func (value *ParameterRef) Validate(ctx context.Context) error {
	if v := value.Value; v != nil {
		return v.Validate(ctx)
	}
	return foundUnresolvedRef(value.Ref)
}

// JSONLookup implements github.com/go-openapi/jsonpointer#JSONPointable
func (value ParameterRef) JSONLookup(token string) (interface{}, error) {
	if token == "$ref" {
		return value.Ref, nil
	}

	ptr, _, err := jsonpointer.GetForToken(value.Value, token)
	return ptr, err
}

// ResponseRef represents either a Response or a $ref to a Response.
// When serializing and both fields are set, Ref is preferred over Value.
type ResponseRef struct {
	Ref   string
	Value *Response
}

var _ jsonpointer.JSONPointable = (*ResponseRef)(nil)

// MarshalJSON returns the JSON encoding of ResponseRef.
func (value *ResponseRef) MarshalJSON() ([]byte, error) {
	return jsoninfo.MarshalRef(value.Ref, value.Value)
}

// UnmarshalJSON sets ResponseRef to a copy of data.
func (value *ResponseRef) UnmarshalJSON(data []byte) error {
	return jsoninfo.UnmarshalRef(data, &value.Ref, &value.Value)
}

// Validate returns an error if ResponseRef does not comply with the OpenAPI spec.
func (value *ResponseRef) Validate(ctx context.Context) error {
	if v := value.Value; v != nil {
		return v.Validate(ctx)
	}
	return foundUnresolvedRef(value.Ref)
}

// JSONLookup implements github.com/go-openapi/jsonpointer#JSONPointable
func (value ResponseRef) JSONLookup(token string) (interface{}, error) {
	if token == "$ref" {
		return value.Ref, nil
	}

	ptr, _, err := jsonpointer.GetForToken(value.Value, token)
	return ptr, err
}

// RequestBodyRef represents either a RequestBody or a $ref to a RequestBody.
// When serializing and both fields are set, Ref is preferred over Value.
type RequestBodyRef struct {
	Ref   string
	Value *RequestBody
}

var _ jsonpointer.JSONPointable = (*RequestBodyRef)(nil)

// MarshalJSON returns the JSON encoding of RequestBodyRef.
func (value *RequestBodyRef) MarshalJSON() ([]byte, error) {
	return jsoninfo.MarshalRef(value.Ref, value.Value)
}

// UnmarshalJSON sets RequestBodyRef to a copy of data.
func (value *RequestBodyRef) UnmarshalJSON(data []byte) error {
	return jsoninfo.UnmarshalRef(data, &value.Ref, &value.Value)
}

// Validate returns an error if RequestBodyRef does not comply with the OpenAPI spec.
func (value *RequestBodyRef) Validate(ctx context.Context) error {
	if v := value.Value; v != nil {
		return v.Validate(ctx)
	}
	return foundUnresolvedRef(value.Ref)
}

// JSONLookup implements github.com/go-openapi/jsonpointer#JSONPointable
func (value RequestBodyRef) JSONLookup(token string) (interface{}, error) {
	if token == "$ref" {
		return value.Ref, nil
	}

	ptr, _, err := jsonpointer.GetForToken(value.Value, token)
	return ptr, err
}

// SchemaRef represents either a Schema or a $ref to a Schema.
// When serializing and both fields are set, Ref is preferred over Value.
type SchemaRef struct {
	Ref   string
	Value *Schema
}

var _ jsonpointer.JSONPointable = (*SchemaRef)(nil)

func NewSchemaRef(ref string, value *Schema) *SchemaRef {
	return &SchemaRef{
		Ref:   ref,
		Value: value,
	}
}

// MarshalJSON returns the JSON encoding of SchemaRef.
func (value *SchemaRef) MarshalJSON() ([]byte, error) {
	return jsoninfo.MarshalRef(value.Ref, value.Value)
}

// UnmarshalJSON sets SchemaRef to a copy of data.
func (value *SchemaRef) UnmarshalJSON(data []byte) error {
	return jsoninfo.UnmarshalRef(data, &value.Ref, &value.Value)
}

// Validate returns an error if SchemaRef does not comply with the OpenAPI spec.
func (value *SchemaRef) Validate(ctx context.Context) error {
	if v := value.Value; v != nil {
		return v.Validate(ctx)
	}
	return foundUnresolvedRef(value.Ref)
}

// JSONLookup implements github.com/go-openapi/jsonpointer#JSONPointable
func (value SchemaRef) JSONLookup(token string) (interface{}, error) {
	if token == "$ref" {
		return value.Ref, nil
	}

	ptr, _, err := jsonpointer.GetForToken(value.Value, token)
	return ptr, err
}

// SecuritySchemeRef represents either a SecurityScheme or a $ref to a SecurityScheme.
// When serializing and both fields are set, Ref is preferred over Value.
type SecuritySchemeRef struct {
	Ref   string
	Value *SecurityScheme
}

var _ jsonpointer.JSONPointable = (*SecuritySchemeRef)(nil)

// MarshalJSON returns the JSON encoding of SecuritySchemeRef.
func (value *SecuritySchemeRef) MarshalJSON() ([]byte, error) {
	return jsoninfo.MarshalRef(value.Ref, value.Value)
}

// UnmarshalJSON sets SecuritySchemeRef to a copy of data.
func (value *SecuritySchemeRef) UnmarshalJSON(data []byte) error {
	return jsoninfo.UnmarshalRef(data, &value.Ref, &value.Value)
}

// Validate returns an error if SecuritySchemeRef does not comply with the OpenAPI spec.
func (value *SecuritySchemeRef) Validate(ctx context.Context) error {
	if v := value.Value; v != nil {
		return v.Validate(ctx)
	}
	return foundUnresolvedRef(value.Ref)
}

// JSONLookup implements github.com/go-openapi/jsonpointer#JSONPointable
func (value SecuritySchemeRef) JSONLookup(token string) (interface{}, error) {
	if token == "$ref" {
		return value.Ref, nil
	}

	ptr, _, err := jsonpointer.GetForToken(value.Value, token)
	return ptr, err
}
