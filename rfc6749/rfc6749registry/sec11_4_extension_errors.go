package rfc6749registry

import (
	"github.com/pkg/errors"
)

// ExtensionError TODO §11.4.1
type ExtensionError struct {
	Name           string
	UsageLocations []ErrorUsageLocation

	RelatedExtension       string
	ChangeController       string
	SpecificationDocuments []string

	Meaning string
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
	String() string
	Meaning() string
}

var authorizationCodeGrantErrorRegistry = make(map[string]AuthorizationCodeGrantError)

type authorizationCodeGrantError struct{ e ExtensionError }

func (e authorizationCodeGrantError) isAuthorizationCodeGrantError() {}
func (e authorizationCodeGrantError) String() string                 { return e.e.Name }
func (e authorizationCodeGrantError) Meaning() string                { return e.e.Meaning }

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
	String() string
	Meaning() string
}

var implicitGrantErrorRegistry = make(map[string]ImplicitGrantError)

type implicitGrantError struct{ e ExtensionError }

func (e implicitGrantError) isImplicitGrantError() {}
func (e implicitGrantError) String() string        { return e.e.Name }
func (e implicitGrantError) Meaning() string       { return e.e.Meaning }

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
	String() string
	Meaning() string
}

var tokenErrorRegistry = make(map[string]TokenError)

type tokenError struct{ e ExtensionError }

func (e tokenError) isTokenError()   {}
func (e tokenError) String() string  { return e.e.Name }
func (e tokenError) Meaning() string { return e.e.Meaning }

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
	String() string
	Meaning() string
}

var resourceAccessErrorRegistry = make(map[string]ResourceAccessError)

type resourceAccessError struct{ e ExtensionError }

func (e resourceAccessError) isResourceAccessError() {}
func (e resourceAccessError) String() string         { return e.e.Name }
func (e resourceAccessError) Meaning() string        { return e.e.Meaning }

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
		authorizationCodeGrantErrorRegistry[e.Name] = authorizationCodeGrantError{e}
	}
	////////////////////////////////////////////////////////////////////////
	if _, set := implicitGrantErrorRegistry[e.Name]; set {
		panic(errors.Errorf("implicit grant error=%q already registered", e.Name))
	}
	if e.usableIn(ImplicitGrantErrorResponse) {
		implicitGrantErrorRegistry[e.Name] = implicitGrantError{e}
	}
	////////////////////////////////////////////////////////////////////////
	if _, set := tokenErrorRegistry[e.Name]; set {
		panic(errors.Errorf("token error=%q already registered", e.Name))
	}
	if e.usableIn(TokenErrorResponse) {
		tokenErrorRegistry[e.Name] = tokenError{e}
	}
	////////////////////////////////////////////////////////////////////////
	if _, set := resourceAccessErrorRegistry[e.Name]; set {
		panic(errors.Errorf("resource access error=%q already registered", e.Name))
	}
	if e.usableIn(ResourceAccessErrorResponse) {
		resourceAccessErrorRegistry[e.Name] = resourceAccessError{e}
	}
}
