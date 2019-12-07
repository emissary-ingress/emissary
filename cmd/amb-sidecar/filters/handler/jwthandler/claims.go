package jwthandler

import (
	jwt "github.com/dgrijalva/jwt-go"

	"github.com/datawire/ambassador/pkg/dlog"

	"github.com/datawire/apro/resourceserver/rfc6749"
)

func GetAudience(claims jwt.MapClaims) []string {
	// https://tools.ietf.org/html/rfc7519#section-4.1.3
	switch claim := claims["aud"].(type) {
	case nil:
		return nil
	case string:
		return []string{claim}
	case []interface{}:
		ret := make([]string, 0, len(claim))
		for _, item := range claim {
			switch item := item.(type) {
			case string:
				ret = append(ret, item)
			default:
				//logger.Warningf("Unexpected aud[n] type: %T", item)
			}
		}
		return ret
	default:
		//logger.Warningf("Unexpected aud type: %T", claim)
		return nil
	}
}

func GetScope(logger dlog.Logger, claims jwt.MapClaims) rfc6749.Scope {
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
