package rfc6749

import (
	"net/http"
	"strings"

	"github.com/pkg/errors"
)

// AccessTokenType stores the registration information for a Access Token Type specified in §11.1.1,
// as well as a Resource Server implementation of the type.
type AccessTokenType struct {
	// Registration metadata
	Name                              string
	AdditionalTokenEndpointParameters []string // TODO(lukeshu): Do something with AccessTokenType.AdditionalTokenEndpointParameters
	ChangeController                  string
	SpecificationDocuments            []string

	// Implementation
	ValidateAuthorization func(*http.Request) (bool, error)
}

// RegisterAccessTokenType registers an Access Token Type with the ResourceServer such that the
// ResourceServer can make use of tokens of that type.
//
// It is a runtime error (panic) to register the same type name multiple times.
func (registry *extensionRegistry) registerAccessTokenType(tokenType AccessTokenType) {
	typeName := strings.ToLower(tokenType.Name)
	if _, set := registry.accessTokenTypes[typeName]; set {
		panic(errors.Errorf("token_type=%q already registered", typeName))
	}
	registry.accessTokenTypes[typeName] = tokenType
}

func (registry *extensionRegistry) GetAccessTokenType(typeName string) (AccessTokenType, bool) {
	registry.ensureInitialized()
	tokenType, ok := registry.accessTokenTypes[strings.ToLower(typeName)]
	return tokenType, ok
}
