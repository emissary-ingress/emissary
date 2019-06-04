package rfc6749

import (
	"github.com/pkg/errors"
)

// AuthorizationEndpointResponseType TODO §11.3.1.
type AuthorizationEndpointResponseType struct {
	Name                   string
	ChangeController       string
	SpecificationDocuments []string
}

var authorizationEndpointResponseTypeRegistry = make(map[string]AuthorizationEndpointResponseType)

// Register TODO
func (responseType AuthorizationEndpointResponseType) Register() {
	if _, set := authorizationEndpointResponseTypeRegistry[responseType.Name]; set {
		panic(errors.Errorf("authorization-endpoint response_type=%q already registered", responseType.Name))
	}
	authorizationEndpointResponseTypeRegistry[responseType.Name] = responseType
}

// GetAuthorizationEndpointResponseType TODO
func GetAuthorizationEndpointResponseType(name string) *AuthorizationEndpointResponseType {
	responseType, ok := authorizationEndpointResponseTypeRegistry[name]
	if !ok {
		return nil
	}
	return &responseType
}

// Initial registry contents, per §11.3.2.
func init() {
	AuthorizationEndpointResponseType{
		Name:                   "code",
		ChangeController:       "IETF",
		SpecificationDocuments: []string{"RFC 6749"},
	}.Register()
	AuthorizationEndpointResponseType{
		Name:                   "token",
		ChangeController:       "IETF",
		SpecificationDocuments: []string{"RFC 6749"},
	}.Register()
}
