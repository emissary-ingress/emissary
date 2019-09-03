package rfc6749

type extensionRegistry struct {
	accessTokenTypes map[string]AccessTokenType // See sec11_1_access_token_types.go
}

// ProtocolExtension stores information about an OAuth protocol extension such that the extension
// can be supported by a Client.  A ProtocolExtension may be added to a Client using the Client's
// `.RegisterProtocolExtension()` method.
type ProtocolExtension struct {
	AccessTokenTypes []AccessTokenType
}

func (registry *extensionRegistry) ensureInitialized() {
	if registry.accessTokenTypes == nil {
		registry.accessTokenTypes = make(map[string]AccessTokenType)
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
	}
}
