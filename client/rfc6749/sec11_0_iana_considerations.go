package rfc6749

type extensionRegistry struct {
	accessTokenTypes map[string]AccessTokenType // See sec11_1_access_token_types.go
}

type ProtocolExtension struct {
	AccessTokenTypes []AccessTokenType
}

func (registry extensionRegistry) RegisterProtocolExtensions(exts ...ProtocolExtension) {
	for _, ext := range exts {
		for _, tokenType := range ext.AccessTokenTypes {
			registry.registerAccessTokenType(tokenType)
		}
	}
}
