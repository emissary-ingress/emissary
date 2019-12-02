package jwthandler

import (
	jwt "github.com/dgrijalva/jwt-go"

	"github.com/datawire/apro/cmd/amb-sidecar/types"
	"github.com/datawire/apro/resourceserver/rfc6749"
)

func GetScope(logger types.Logger, claims jwt.MapClaims) rfc6749.Scope {
	// Parse the 'scope' claim (draft standard).
	// https://www.iana.org/assignments/jwt/jwt.xhtml
	// https://tools.ietf.org/html/draft-ietf-oauth-token-exchange-19#section-4.2
	switch scopeClaim := claims["scope"].(type) {
	case nil:
		logger.Debugf("No scope")
		return nil
	case string: // proposed standard; most Authorization Servers do this
		return rfc6749.ParseScope(scopeClaim)
	case []interface{}: // UAA does this
		actual := make(rfc6749.Scope, len(scopeClaim))
		for _, scopeValue := range scopeClaim {
			switch scopeValue := scopeValue.(type) {
			case string:
				actual[scopeValue] = struct{}{}
			default:
				logger.Warningf("Unexpected scope[n] type: %T", scopeValue)
			}
		}
		return actual
	default:
		logger.Warningf("Unexpected scope type: %T", scopeClaim)
		return nil
	}
}
