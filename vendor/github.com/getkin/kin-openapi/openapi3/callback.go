package openapi3

import (
	"context"
	"fmt"

	"github.com/go-openapi/jsonpointer"
)

type Callbacks map[string]*CallbackRef

var _ jsonpointer.JSONPointable = (*Callbacks)(nil)

// JSONLookup implements github.com/go-openapi/jsonpointer#JSONPointable
func (c Callbacks) JSONLookup(token string) (interface{}, error) {
	ref, ok := c[token]
	if ref == nil || !ok {
		return nil, fmt.Errorf("object has no field %q", token)
	}

	if ref.Ref != "" {
		return &Ref{Ref: ref.Ref}, nil
	}
	return ref.Value, nil
}

// Callback is specified by OpenAPI/Swagger standard version 3.
// See https://github.com/OAI/OpenAPI-Specification/blob/main/versions/3.0.3.md#callbackObject
type Callback map[string]*PathItem

// Validate returns an error if Callback does not comply with the OpenAPI spec.
func (callback Callback) Validate(ctx context.Context) error {
	for _, v := range callback {
		if err := v.Validate(ctx); err != nil {
			return err
		}
	}
	return nil
}
