package rfc6749

import (
	"io"
	"net/http"
	"strings"

	"github.com/pkg/errors"
)

// AccessTokenType stores the registration information for a Access
// Token Type specified in §11.1.1, as well as a Client implementation
// of the type.
type AccessTokenType struct {
	// Registration metadata
	Name                              string
	AdditionalTokenEndpointParameters []string // TODO(lukeshu): Will definitely need to do something with AccessTokenType.AdditionalTokenEndpointParameters
	ChangeController                  string
	SpecificationDocuments            []string

	// Implementation
	NeedsBody                       bool // Whether AuthorizationForResourceRequest needs the resource request body
	AuthorizationForResourceRequest func(token string, body io.Reader) (http.Header, error)
}

type accessTokenTypeRegistry map[string]AccessTokenType

// RegisterAccessTokenType registers an Access Token Type with the
// Client such that the Client can make use of tokens of that type.
//
// It is a runtime error (panic) to register the same type name
// multiple times.
func (registry accessTokenTypeRegistry) RegisterAccessTokenType(tokenType AccessTokenType) {
	typeName := strings.ToLower(tokenType.Name)
	if _, set := registry[typeName]; set {
		panic(errors.Errorf("token_type=%q already registered", typeName))
	}
	registry[typeName] = tokenType
}

func (registry accessTokenTypeRegistry) getAccessTokenType(typeName string) (AccessTokenType, bool) {
	tokenType, ok := registry[strings.ToLower(typeName)]
	return tokenType, ok
}
