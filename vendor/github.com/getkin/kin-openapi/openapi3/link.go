package openapi3

import (
	"context"
	"errors"
	"fmt"

	"github.com/go-openapi/jsonpointer"

	"github.com/getkin/kin-openapi/jsoninfo"
)

type Links map[string]*LinkRef

// JSONLookup implements github.com/go-openapi/jsonpointer#JSONPointable
func (links Links) JSONLookup(token string) (interface{}, error) {
	ref, ok := links[token]
	if ok == false {
		return nil, fmt.Errorf("object has no field %q", token)
	}

	if ref != nil && ref.Ref != "" {
		return &Ref{Ref: ref.Ref}, nil
	}
	return ref.Value, nil
}

var _ jsonpointer.JSONPointable = (*Links)(nil)

// Link is specified by OpenAPI/Swagger standard version 3.
// See https://github.com/OAI/OpenAPI-Specification/blob/main/versions/3.0.3.md#linkObject
type Link struct {
	ExtensionProps

	OperationRef string                 `json:"operationRef,omitempty" yaml:"operationRef,omitempty"`
	OperationID  string                 `json:"operationId,omitempty" yaml:"operationId,omitempty"`
	Description  string                 `json:"description,omitempty" yaml:"description,omitempty"`
	Parameters   map[string]interface{} `json:"parameters,omitempty" yaml:"parameters,omitempty"`
	Server       *Server                `json:"server,omitempty" yaml:"server,omitempty"`
	RequestBody  interface{}            `json:"requestBody,omitempty" yaml:"requestBody,omitempty"`
}

// MarshalJSON returns the JSON encoding of Link.
func (link *Link) MarshalJSON() ([]byte, error) {
	return jsoninfo.MarshalStrictStruct(link)
}

// UnmarshalJSON sets Link to a copy of data.
func (link *Link) UnmarshalJSON(data []byte) error {
	return jsoninfo.UnmarshalStrictStruct(data, link)
}

// Validate returns an error if Link does not comply with the OpenAPI spec.
func (link *Link) Validate(ctx context.Context) error {
	if link.OperationID == "" && link.OperationRef == "" {
		return errors.New("missing operationId or operationRef on link")
	}
	if link.OperationID != "" && link.OperationRef != "" {
		return fmt.Errorf("operationId %q and operationRef %q are mutually exclusive", link.OperationID, link.OperationRef)
	}
	return nil
}
