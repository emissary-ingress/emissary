package rfc6749registry

import (
	"github.com/pkg/errors"
)

// ExtensionError TODO §11.4.1
type ExtensionError struct {
	Name                   string
	UsageLocations         []ErrorUsageLocation
	RelatedExtension       string
	ChangeController       string
	SpecificationDocuments []string
}

// ErrorUsageLocation TODO §11.4.1
type ErrorUsageLocation interface {
	isErrorUsageLocation()
}

type errorUsageLocation string

func (l errorUsageLocation) isErrorUsageLocation() {}

func newErrorUsageLocation(s string) ErrorUsageLocation {
	return errorUsageLocation(s)
}

// TODO §11.4.1
var (
	AuthorizationCodeGrantErrorResponse = newErrorUsageLocation("authorization code grant error response") // §4.1.2.1
	ImplicitGrantErrorResponse          = newErrorUsageLocation("implicit grant error response")           // §4.2.2.1
	TokenErrorResponse                  = newErrorUsageLocation("token error response")                    // §5.2
	ResourceAccessErrorResponse         = newErrorUsageLocation("resource access error response")          // §7.2
)

////////////////////////////////////////////////////////////////////////////////

type AuthorizationCodeGrantError interface {
	isAuthorizationCodeGrantError()
}

var authorizationCodeGrantErrorRegistry = make(map[string]AuthorizationCodeGrantError)

type authorizationCodeGrantError ExtensionError

func (e authorizationCodeGrantError) isAuthorizationCodeGrantError() {}

// GetAuthorizationCodeGrantError TODO
func GetAuthorizationCodeGrantError(name string) AuthorizationCodeGrantError {
	ecode, ok := authorizationCodeGrantErrorRegistry[name]
	if !ok {
		return nil
	}
	return ecode
}

////////////////////////////////////////////////////////////////////////////////

type ImplicitGrantError interface {
	isImplicitGrantError()
}

var implicitGrantErrorRegistry = make(map[string]ImplicitGrantError)

type implicitGrantError ExtensionError

func (e implicitGrantError) isImplicitGrantError() {}

// GetImplicitGrantError TODO
func GetImplicitGrantError(name string) ImplicitGrantError {
	ecode, ok := implicitGrantErrorRegistry[name]
	if !ok {
		return nil
	}
	return ecode
}

////////////////////////////////////////////////////////////////////////////////

type TokenError interface {
	isTokenError()
}

var tokenErrorRegistry = make(map[string]TokenError)

type tokenError ExtensionError

func (e tokenError) isTokenError() {}

// GetTokenError TODO
func GetTokenError(name string) TokenError {
	ecode, ok := tokenErrorRegistry[name]
	if !ok {
		return nil
	}
	return ecode
}

////////////////////////////////////////////////////////////////////////////////

type ResourceAccessError interface {
	isResourceAccessError()
}

var resourceAccessErrorRegistry = make(map[string]ResourceAccessError)

type resourceAccessError ExtensionError

func (e resourceAccessError) isResourceAccessError() {}

// GetResourceAccessError TODO
func GetResourceAccessError(name string) ResourceAccessError {
	ecode, ok := resourceAccessErrorRegistry[name]
	if !ok {
		return nil
	}
	return ecode
}

////////////////////////////////////////////////////////////////////////////////

func (e ExtensionError) usableIn(loc ErrorUsageLocation) bool {
	for _, l := range e.UsageLocations {
		if l == loc {
			return true
		}
	}
	return false
}

func (e ExtensionError) Register() {
	////////////////////////////////////////////////////////////////////////
	if _, set := authorizationCodeGrantErrorRegistry[e.Name]; set {
		panic(errors.Errorf("authorization code grant error=%q already registered", e.Name))
	}
	if e.usableIn(AuthorizationCodeGrantErrorResponse) {
		authorizationCodeGrantErrorRegistry[e.Name] = authorizationCodeGrantError(e)
	}
	////////////////////////////////////////////////////////////////////////
	if _, set := implicitGrantErrorRegistry[e.Name]; set {
		panic(errors.Errorf("implicit grant error=%q already registered", e.Name))
	}
	if e.usableIn(ImplicitGrantErrorResponse) {
		implicitGrantErrorRegistry[e.Name] = implicitGrantError(e)
	}
	////////////////////////////////////////////////////////////////////////
	if _, set := tokenErrorRegistry[e.Name]; set {
		panic(errors.Errorf("token error=%q already registered", e.Name))
	}
	if e.usableIn(TokenErrorResponse) {
		tokenErrorRegistry[e.Name] = tokenError(e)
	}
	////////////////////////////////////////////////////////////////////////
	if _, set := resourceAccessErrorRegistry[e.Name]; set {
		panic(errors.Errorf("resource access error=%q already registered", e.Name))
	}
	if e.usableIn(ResourceAccessErrorResponse) {
		resourceAccessErrorRegistry[e.Name] = resourceAccessError(e)
	}
}
