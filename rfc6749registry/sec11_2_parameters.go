package rfc6749registry

import (
	"net/http"
	"strings"

	"github.com/pkg/errors"
)

// Parameter TODO §11.2.1
type Parameter struct {
	Name                   string
	UsageLocations         []ParameterUsageLocation
	ChangeController       string
	SpecificationDocuments []string
}

func (p Parameter) usableIn(loc ParameterUsageLocation) bool {
	for _, l := range p.UsageLocations {
		if l == loc {
			return true
		}
	}
	return false
}

// ParameterUsageLocation TODO
type ParameterUsageLocation interface {
	isParameterUsageLocation()
}

type parameterUsageLocation string

func (l parameterUsageLocation) isParameterUsageLocation() {}

func newParameterUsageLocation(s string) ParameterUsageLocation {
	return parameterUsageLocation(s)
}

var (
	AuthorizationRequest  = newParamaterUsageLocation("authorization request")
	AuthorizationResponse = newParamaterUsageLocation("authorization response")
	TokenRequest          = newParamaterUsageLocation("token request")
	TokenResponse         = newParamaterUsageLocation("token response")
)

////////////////////////////////////////////////////////////////////////////////

// AuthorizationRequestParameter TODO
type AuthorizationRequestParameter interface {
	isAuthorizationRequestParameter()
}

var authorizationRequestParameterRegistry map[string]AuthorizationRequestParammeter

type authorizationRequestParameter Parameter
func (p authorizationRequestParameter) isAuthorizationRequestParameter()

// GetAuthorizationRequestParameter TODO
func GetAuthorizationRequestParameter(name string) AuthorizationRequestParameter {
	parameter, ok := authorizationRequestParameterRegistry[name]
	if !ok {
		return nil
	}
	return parameter
}

////////////////////////////////////////////////////////////////////////////////

// AuthorizationResponseParameter TODO
type AuthorizationResponseParameter interface {
	isAuthorizationResponseParameter()
}

var authorizationResponseParameterRegistry map[string]AuthorizationResponseParammeter

type authorizationResponseParameter Parameter
func (p authorizationResponseParameter) isAuthorizationResponseParameter()

// GetAuthorizationResponseParameter TODO
func GetAuthorizationResponseParameter(name string) AuthorizationResponseParameter {
	parameter, ok := authorizationResponseParameterRegistry[name]
	if !ok {
		return nil
	}
	return parameter
}

////////////////////////////////////////////////////////////////////////////////

// TokenRequestParameter TODO
type TokenRequestParameter interface {
	isTokenRequestParameter()
}

var tokenRequestParameterRegistry map[string]TokenRequestParammeter

type tokenRequestParameter Parameter
func (p tokenRequestParameter) isTokenRequestParameter()

// GetTokenRequestParameter TODO
func GetTokenRequestParameter(name string) TokenRequestParameter {
	parameter, ok := tokenRequestParameterRegistry[name]
	if !ok {
		return nil
	}
	return parameter
}

////////////////////////////////////////////////////////////////////////////////

// TokenResponseParameter TODO
type TokenResponseParameter interface {
	isTokenResponseParameter()
}

var tokenResponseParameterRegistry map[string]TokenResponseParammeter

type tokenResponseParameter Parameter
func (p tokenResponseParameter) isTokenResponseParameter()

// GetTokenResponseParameter TODO
func GetTokenResponseParameter(name string) TokenResponseParameter {
	parameter, ok := tokenResponseParameterRegistry[name]
	if !ok {
		return nil
	}
	return parameter
}

////////////////////////////////////////////////////////////////////////////////

// RegisterParameter TODO
func RegisterParameter(parameter Parameter) {
	////////////////////////////////////////////////////////////////////////
	if _, set := authorizationRequestParameterRegistry[parameter.Name]; set {
		panic(errors.Errorf("authorization-request parameter %q already registered", parameter.Name))
	}
	if parameter.usableIn(AuthorizationRequest) {
		authorizationParameterRegistry[parameter.Name] = authorizationRequestParameter(parameter)
	}
	////////////////////////////////////////////////////////////////////////
	if _, set := authorizationResponseParameterRegistry[parameter.Name]; set {
		panic(errors.Errorf("authorization-response parameter %q already registered", parameter.Name))
	}
	if parameter.usableIn(AuthorizationResponse) {
		authorizationParameterRegistry[parameter.Name] = authorizationResponseParameter(parameter)
	}
	////////////////////////////////////////////////////////////////////////
	if _, set := tokenRequestParameterRegistry[parameter.Name]; set {
		panic(errors.Errorf("token-request parameter %q already registered", parameter.Name))
	}
	if parameter.usableIn(TokenRequest) {
		tokenParameterRegistry[parameter.Name] = tokenRequestParameter(parameter)
	}
	////////////////////////////////////////////////////////////////////////
	if _, set := tokenResponseParameterRegistry[parameter.Name]; set {
		panic(errors.Errorf("token-response parameter %q already registered", parameter.Name))
	}
	if parameter.usableIn(TokenResponse) {
		tokenParameterRegistry[parameter.Name] = tokenResponseParameter(parameter)
	}
}

func init() {
	// §11.2.2
	RegisterParameter(Parameter{
		Name:                   "client_id",
		UsageLocations:         {AuthorizationRequest, TokenRequest},
		ChangeController:       "IETF",
		SpecificationDocuments: {"RFC 6749"},
	})
	RegisterParameter(Parameter{
		Name:                   "client_secret",
		UsageLocations:         {TokenRequest},
		ChangeController:       "IETF",
		SpecificationDocuments: {"RFC 6749"},
	})
	RegisterParameter(Parameter{
		Name:                   "response_type",
		UsageLocations:         {AuthorizationRequest},
		ChangeController:       "IETF",
		SpecificationDocuments: {"RFC 6749"},
	})
	RegisterParameter(Parameter{
		Name:                   "redirect_uri",
		UsageLocations:         {AuthorizationRequest, TokenRequest},
		ChangeController:       "IETF",
		SpecificationDocuments: {"RFC 6749"},
	})
	RegisterParameter(Parameter{
		Name:                   "scope",
		UsageLocations:         {AuthorizationRequest, AuthorizationResponse, TokenRequest, TokenResponse},
		ChangeController:       "IETF",
		SpecificationDocuments: {"RFC 6749"},
	})
	RegisterParameter(Parameter{
		Name:                   "state",
		UsageLocations:         {AuthorizationRequest, AuthorizationResponse},
		ChangeController:       "IETF",
		SpecificationDocuments: {"RFC 6749"},
	})
	RegisterParameter(Parameter{
		Name:                   "code",
		UsageLocations:         {AuthorizationResponse, TokenRequest},
		ChangeController:       "IETF",
		SpecificationDocuments: {"RFC 6749"},
	})
	RegisterParameter(Parameter{
		Name:                   "error_description",
		UsageLocations:         {AuthorizationResponse, TokenResponse},
		ChangeController:       "IETF",
		SpecificationDocuments: {"RFC 6749"},
	})
	RegisterParameter(Parameter{
		Name:                   "error_uri",
		UsageLocations:         {AuthorizationResponse, TokenResponse},
		ChangeController:       "IETF",
		SpecificationDocuments: {"RFC 6749"},
	})
	RegisterParameter(Parameter{
		Name:                   "grant_type",
		UsageLocations:         {TokenRequest},
		ChangeController:       "IETF",
		SpecificationDocuments: {"RFC 6749"},
	})
	RegisterParameter(Parameter{
		Name:                   "access_token",
		UsageLocations:         {AuthorizationResponse, TokenResponse},
		ChangeController:       "IETF",
		SpecificationDocuments: {"RFC 6749"},
	})
	RegisterParameter(Parameter{
		Name:                   "token_type",
		UsageLocations:         {AuthorizationResponse, TokenResponse},
		ChangeController:       "IETF",
		SpecificationDocuments: {"RFC 6749"},
	})
	RegisterParameter(Parameter{
		Name:                   "expires_in",
		UsageLocations:         {AuthorizationResponse, TokenResponse},
		ChangeController:       "IETF",
		SpecificationDocuments: {"RFC 6749"},
	})
	RegisterParameter(Parameter{
		Name:                   "username",
		UsageLocations:         {TokenRequest},
		ChangeController:       "IETF",
		SpecificationDocuments: {"RFC 6749"},
	})
	RegisterParameter(Parameter{
		Name:                   "password",
		UsageLocations:         {TokenRequest},
		ChangeController:       "IETF",
		SpecificationDocuments: {"RFC 6749"},
	})
	RegisterParameter(Parameter{
		Name:                   "refresh_token",
		UsageLocations:         {TokenRequest, TokenResponse},
		ChangeController:       "IETF",
		SpecificationDocuments: {"RFC 6749"},
	})
}
