package rfc6749

type extensionRegistry struct {
	// See sec11_1_access_token_types.go
	accessTokenTypes map[string]AccessTokenType

	// See sec11_4_extensions_error.go
	authorizationCodeGrantErrors map[string]ExtensionError
	implicitGrantErrors          map[string]ExtensionError
	tokenErrors                  map[string]ExtensionError
	resourceAccessErrors         map[string]ExtensionError
}

// ProtocolExtension stores information about an OAuth protocol extension such that the extension
// can be supported by a Client.  A ProtocolExtension may be added to a Client using the Client's
// `.RegisterProtocolExtension()` method.
type ProtocolExtension struct {
	AccessTokenTypes []AccessTokenType
	ExtensionErrors  []ExtensionError
}

func (registry *extensionRegistry) ensureInitialized() {
	if registry.accessTokenTypes == nil {
		registry.accessTokenTypes = make(map[string]AccessTokenType)
	}
	if registry.authorizationCodeGrantErrors == nil {
		registry.authorizationCodeGrantErrors = newBuiltInAuthorizationCodeGrantErrors()
	}
	if registry.implicitGrantErrors == nil {
		registry.implicitGrantErrors = newBuiltInImplicitGrantErrors()
	}
	if registry.tokenErrors == nil {
		registry.tokenErrors = newBuiltInTokenErrors()
	}
	if registry.resourceAccessErrors == nil {
		registry.resourceAccessErrors = make(map[string]ExtensionError)
	}
}

// RegisterProtocolExtensions adds support for an OAuth ProtocolExtension to the Client.
//
// It is a runtime error (panic) to register the conficting extensions or to register the same
// extension multiple times.
func (registry *extensionRegistry) RegisterProtocolExtensions(exts ...ProtocolExtension) {
	registry.ensureInitialized()
	for _, ext := range exts {
		for _, tokenType := range ext.AccessTokenTypes {
			registry.registerAccessTokenType(tokenType)
		}
		for _, err := range ext.ExtensionErrors {
			registry.registerError(err)
		}
	}
}
