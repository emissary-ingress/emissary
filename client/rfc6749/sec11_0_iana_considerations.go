package rfc6749

type extensionRegistry struct {
	accessTokenTypes map[string]AccessTokenType // See sec11_1_access_token_types.go
}

// ProtocolExtension stores information about an OAuth protocol extension such that the extensionc
// an be supported by a Client.  A ProtocolExtension may be added to a Client using the Client's
// `.RegisterProtocolExtension()` method.
type ProtocolExtension struct {
	AccessTokenTypes []AccessTokenType
}

// RegisterProtocolExtensions adds support for an OAuth ProtocolExtension to the Client.
func (registry extensionRegistry) RegisterProtocolExtensions(exts ...ProtocolExtension) {
	for _, ext := range exts {
		for _, tokenType := range ext.AccessTokenTypes {
			registry.registerAccessTokenType(tokenType)
		}
	}
}
