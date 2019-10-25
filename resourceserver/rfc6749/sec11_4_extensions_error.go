package rfc6749

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
