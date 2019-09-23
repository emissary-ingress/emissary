package rfc6749

import (
	"github.com/pkg/errors"
)

// Parameter TODO §11.2.1
type Parameter struct {
	Name                   string
	UsageLocations         []ParameterUsageLocation
	ChangeController       string
	SpecificationDocuments []string
}

// ParameterUsageLocation TODO §11.2.1
type ParameterUsageLocation interface {
	isParameterUsageLocation()
}

type parameterUsageLocation string

func (l parameterUsageLocation) isParameterUsageLocation() {}

func newParameterUsageLocation(s string) ParameterUsageLocation {
	return parameterUsageLocation(s)
}

// TODO §11.2.1
var (
	AuthorizationRequest  = newParameterUsageLocation("authorization request")
	AuthorizationResponse = newParameterUsageLocation("authorization response")
	TokenRequest          = newParameterUsageLocation("token request")
	TokenResponse         = newParameterUsageLocation("token response")
)

////////////////////////////////////////////////////////////////////////////////

// AuthorizationRequestParameter TODO
type AuthorizationRequestParameter interface {
	isAuthorizationRequestParameter()
}

var authorizationRequestParameterRegistry = make(map[string]AuthorizationRequestParameter)

type authorizationRequestParameter Parameter

func (p authorizationRequestParameter) isAuthorizationRequestParameter() {}

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

var authorizationResponseParameterRegistry = make(map[string]AuthorizationResponseParameter)

type authorizationResponseParameter Parameter

func (p authorizationResponseParameter) isAuthorizationResponseParameter() {}

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

var tokenRequestParameterRegistry = make(map[string]TokenRequestParameter)

type tokenRequestParameter Parameter

func (p tokenRequestParameter) isTokenRequestParameter() {}

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

var tokenResponseParameterRegistry = make(map[string]TokenResponseParameter)

type tokenResponseParameter Parameter

func (p tokenResponseParameter) isTokenResponseParameter() {}

// GetTokenResponseParameter TODO
func GetTokenResponseParameter(name string) TokenResponseParameter {
	parameter, ok := tokenResponseParameterRegistry[name]
	if !ok {
		return nil
	}
	return parameter
}

////////////////////////////////////////////////////////////////////////////////

func (parameter Parameter) usableIn(loc ParameterUsageLocation) bool {
	for _, l := range parameter.UsageLocations {
		if l == loc {
			return true
		}
	}
	return false
}

// Register TODO
func (parameter Parameter) Register() {
	////////////////////////////////////////////////////////////////////////
	if _, set := authorizationRequestParameterRegistry[parameter.Name]; set {
		panic(errors.Errorf("authorization-request parameter %q already registered", parameter.Name))
	}
	if parameter.usableIn(AuthorizationRequest) {
		authorizationRequestParameterRegistry[parameter.Name] = authorizationRequestParameter(parameter)
	}
	////////////////////////////////////////////////////////////////////////
	if _, set := authorizationResponseParameterRegistry[parameter.Name]; set {
		panic(errors.Errorf("authorization-response parameter %q already registered", parameter.Name))
	}
	if parameter.usableIn(AuthorizationResponse) {
		authorizationResponseParameterRegistry[parameter.Name] = authorizationResponseParameter(parameter)
	}
	////////////////////////////////////////////////////////////////////////
	if _, set := tokenRequestParameterRegistry[parameter.Name]; set {
		panic(errors.Errorf("token-request parameter %q already registered", parameter.Name))
	}
	if parameter.usableIn(TokenRequest) {
		tokenRequestParameterRegistry[parameter.Name] = tokenRequestParameter(parameter)
	}
	////////////////////////////////////////////////////////////////////////
	if _, set := tokenResponseParameterRegistry[parameter.Name]; set {
		panic(errors.Errorf("token-response parameter %q already registered", parameter.Name))
	}
	if parameter.usableIn(TokenResponse) {
		tokenResponseParameterRegistry[parameter.Name] = tokenResponseParameter(parameter)
	}
}

// Initial registry contents, per §11.2.2.
func init() {
	Parameter{
		Name:                   "client_id",
		UsageLocations:         []ParameterUsageLocation{AuthorizationRequest, TokenRequest},
		ChangeController:       "IETF",
		SpecificationDocuments: []string{"RFC 6749"},
	}.Register()
	Parameter{
		Name:                   "client_secret",
		UsageLocations:         []ParameterUsageLocation{TokenRequest},
		ChangeController:       "IETF",
		SpecificationDocuments: []string{"RFC 6749"},
	}.Register()
	Parameter{
		Name:                   "response_type",
		UsageLocations:         []ParameterUsageLocation{AuthorizationRequest},
		ChangeController:       "IETF",
		SpecificationDocuments: []string{"RFC 6749"},
	}.Register()
	Parameter{
		Name:                   "redirect_uri",
		UsageLocations:         []ParameterUsageLocation{AuthorizationRequest, TokenRequest},
		ChangeController:       "IETF",
		SpecificationDocuments: []string{"RFC 6749"},
	}.Register()
	Parameter{
		Name:                   "scope",
		UsageLocations:         []ParameterUsageLocation{AuthorizationRequest, AuthorizationResponse, TokenRequest, TokenResponse},
		ChangeController:       "IETF",
		SpecificationDocuments: []string{"RFC 6749"},
	}.Register()
	Parameter{
		Name:                   "state",
		UsageLocations:         []ParameterUsageLocation{AuthorizationRequest, AuthorizationResponse},
		ChangeController:       "IETF",
		SpecificationDocuments: []string{"RFC 6749"},
	}.Register()
	Parameter{
		Name:                   "code",
		UsageLocations:         []ParameterUsageLocation{AuthorizationResponse, TokenRequest},
		ChangeController:       "IETF",
		SpecificationDocuments: []string{"RFC 6749"},
	}.Register()
	Parameter{
		Name:                   "error_description",
		UsageLocations:         []ParameterUsageLocation{AuthorizationResponse, TokenResponse},
		ChangeController:       "IETF",
		SpecificationDocuments: []string{"RFC 6749"},
	}.Register()
	Parameter{
		Name:                   "error_uri",
		UsageLocations:         []ParameterUsageLocation{AuthorizationResponse, TokenResponse},
		ChangeController:       "IETF",
		SpecificationDocuments: []string{"RFC 6749"},
	}.Register()
	Parameter{
		Name:                   "grant_type",
		UsageLocations:         []ParameterUsageLocation{TokenRequest},
		ChangeController:       "IETF",
		SpecificationDocuments: []string{"RFC 6749"},
	}.Register()
	Parameter{
		Name:                   "access_token",
		UsageLocations:         []ParameterUsageLocation{AuthorizationResponse, TokenResponse},
		ChangeController:       "IETF",
		SpecificationDocuments: []string{"RFC 6749"},
	}.Register()
	Parameter{
		Name:                   "token_type",
		UsageLocations:         []ParameterUsageLocation{AuthorizationResponse, TokenResponse},
		ChangeController:       "IETF",
		SpecificationDocuments: []string{"RFC 6749"},
	}.Register()
	Parameter{
		Name:                   "expires_in",
		UsageLocations:         []ParameterUsageLocation{AuthorizationResponse, TokenResponse},
		ChangeController:       "IETF",
		SpecificationDocuments: []string{"RFC 6749"},
	}.Register()
	Parameter{
		Name:                   "username",
		UsageLocations:         []ParameterUsageLocation{TokenRequest},
		ChangeController:       "IETF",
		SpecificationDocuments: []string{"RFC 6749"},
	}.Register()
	Parameter{
		Name:                   "password",
		UsageLocations:         []ParameterUsageLocation{TokenRequest},
		ChangeController:       "IETF",
		SpecificationDocuments: []string{"RFC 6749"},
	}.Register()
	Parameter{
		Name:                   "refresh_token",
		UsageLocations:         []ParameterUsageLocation{TokenRequest, TokenResponse},
		ChangeController:       "IETF",
		SpecificationDocuments: []string{"RFC 6749"},
	}.Register()
}
