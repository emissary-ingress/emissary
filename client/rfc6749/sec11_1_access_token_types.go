package rfc6749

import (
	"io"
	"net/http"
	"strings"

	"github.com/pkg/errors"
)

// AccessTokenType stores the registration information for a Access Token Type specified in §11.1.1,
// as well as a Client implementation of the type.
type AccessTokenType struct {
	// Registration metadata
	Name                              string
	AdditionalTokenEndpointParameters []string // TODO(lukeshu): Do something with AccessTokenType.AdditionalTokenEndpointParameters
	ChangeController                  string
	SpecificationDocuments            []string

	// Implementation
	NeedsBody                       bool // Whether AuthorizationForResourceRequest needs the resource-request body
	AuthorizationForResourceRequest func(token string, body io.Reader) (http.Header, error)
}

// registerAccessTokenType registers an Access Token Type with the Client such that the Client can
// make use of tokens of that type.
//
// It is a runtime error (panic) to register the same type name multiple times.
func (registry *extensionRegistry) registerAccessTokenType(tokenType AccessTokenType) {
	typeName := strings.ToLower(tokenType.Name)
	if _, set := registry.accessTokenTypes[typeName]; set {
		panic(errors.Errorf("token_type=%q already registered", typeName))
	}
	registry.accessTokenTypes[typeName] = tokenType
}

func (registry *extensionRegistry) getAccessTokenType(typeName string) (AccessTokenType, bool) {
	registry.ensureInitialized()
	tokenType, ok := registry.accessTokenTypes[strings.ToLower(typeName)]
	return tokenType, ok
}
