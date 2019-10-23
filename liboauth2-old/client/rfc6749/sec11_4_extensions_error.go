package rfc6749

import (
	"github.com/pkg/errors"
)

// ExtensionError stores the registration information for an Extension Error specified in §11.4.1,
// as well as implementation-data on the error meaning.
type ExtensionError struct {
	// Registration metadata
	Name                   string
	UsageLocations         []ErrorUsageLocation
	RelatedExtension       string
	ChangeController       string
	SpecificationDocuments []string

	// Implementation
	Meaning string
}

// ErrorUsageLocation is an enum of the locations that an error code may appear in, as enumerated in
// §11.4.1.
type ErrorUsageLocation interface {
	isErrorUsageLocation()
}

type errorUsageLocation string

func (l errorUsageLocation) isErrorUsageLocation() {}

func newErrorUsageLocation(s string) ErrorUsageLocation {
	return errorUsageLocation(s)
}

// These are the locations that an error code may appear in, as enumerated in §11.4.1.
var (
	LocationAuthorizationCodeGrantErrorResponse = newErrorUsageLocation("authorization code grant error response") // §4.1.2.1
	LocationImplicitGrantErrorResponse          = newErrorUsageLocation("implicit grant error response")           // §4.2.2.1
	LocationTokenErrorResponse                  = newErrorUsageLocation("token error response")                    // §5.2
	LocationResourceAccessErrorResponse         = newErrorUsageLocation("resource access error response")          // §7.2
)

////////////////////////////////////////////////////////////////////////////////

func (e ExtensionError) usableIn(loc ErrorUsageLocation) bool {
	for _, l := range e.UsageLocations {
		if l == loc {
			return true
		}
	}
	return false
}

// registerError registers an Error with the Client such that the Client can understand
// that error code, and provide helpful output.
//
// It is a runtime error (panic) to register the same error name with the same location type name
// multiple times.
func (registry *extensionRegistry) registerError(e ExtensionError) {
	////////////////////////////////////////////////////////////////////////
	if e.usableIn(LocationAuthorizationCodeGrantErrorResponse) {
		if _, set := registry.authorizationCodeGrantErrors[e.Name]; set {
			panic(errors.Errorf("authorization code grant error=%q already registered", e.Name))
		}
		registry.authorizationCodeGrantErrors[e.Name] = e
	}
	////////////////////////////////////////////////////////////////////////
	if e.usableIn(LocationImplicitGrantErrorResponse) {
		if _, set := registry.implicitGrantErrors[e.Name]; set {
			panic(errors.Errorf("implicit grant error=%q already registered", e.Name))
		}
		registry.implicitGrantErrors[e.Name] = e
	}
	////////////////////////////////////////////////////////////////////////
	if e.usableIn(LocationTokenErrorResponse) {
		if _, set := registry.tokenErrors[e.Name]; set {
			panic(errors.Errorf("token error=%q already registered", e.Name))
		}
		registry.tokenErrors[e.Name] = e
	}
	////////////////////////////////////////////////////////////////////////
	if e.usableIn(LocationResourceAccessErrorResponse) {
		if _, set := registry.resourceAccessErrors[e.Name]; set {
			panic(errors.Errorf("resource access error=%q already registered", e.Name))
		}
		registry.resourceAccessErrors[e.Name] = e
	}
}
