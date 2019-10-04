package oauth2handler

import (
	"context"
	"net/http"

	jwt "github.com/dgrijalva/jwt-go"
	"github.com/pkg/errors"

	rfc6749common "github.com/datawire/liboauth2/common/rfc6749"
	rfc6750resourceserver "github.com/datawire/liboauth2/resourceserver/rfc6750"

	"github.com/datawire/apro/cmd/amb-sidecar/filters/handler/middleware"
	"github.com/datawire/apro/cmd/amb-sidecar/types"
	"github.com/datawire/apro/lib/filterapi"
	"github.com/datawire/apro/lib/filterapi/filterutil"
	"github.com/datawire/apro/lib/jwtsupport"
)

// filterResourceServer implements the OAuth Resource Server part of the Filter.
//
// As a "special" case, a FilterResponse of "nil" means to send the
// same request to the upstream service (the other half of the
// Resource Server).
func (rs *OAuth2Filter) filterResourceServer(ctx context.Context, logger types.Logger, httpClient *http.Client, discovered *Discovered, request *filterapi.FilterRequest, scope rfc6749common.Scope) filterapi.FilterResponse {
	// Validate the scope values we were granted.  We take the scope as an
	// argument, instead of extracting it from the authorization, because there
	// isn't actually a good portable way to extract it from the authorization.
	if err := rs.validateScope(scope); err != nil {
		return middleware.NewErrorResponse(ctx, http.StatusForbidden,
			errors.Wrap(err, "insufficient privilege scope"), nil)
	}
	// Validate the authorization.
	token := rfc6750resourceserver.GetFromHeader(filterutil.GetHeader(request))
	if err := rs.validateAccessToken(token, discovered, httpClient, logger); err != nil {
		return middleware.NewErrorResponse(ctx, http.StatusBadRequest, err, nil)
	}
	// If everything has passed, go ahead and have Envoy proxy to the other half
	// of the Resource Server.
	return nil
}

func (rs *OAuth2Filter) validateAccessToken(token string, discovered *Discovered, httpClient *http.Client, logger types.Logger) error {
	switch rs.Spec.AccessTokenValidation {
	case "auto":
		claims, err := rs.parseJWT(token, discovered)
		if err == nil {
			return rs.validateJWT(claims, discovered, logger)
		}
		logger.Debugln("rejecting JWT validation; falling back to UserInfo Endpoint validation:", err)
		fallthrough
	case "userinfo":
		return rs.validateAccessTokenUserinfo(token, discovered, httpClient, logger)
	case "jwt":
		claims, err := rs.parseJWT(token, discovered)
		if err != nil {
			return err
		}
		return rs.validateJWT(claims, discovered, logger)
	}
	panic("not reached")
}

func (rs *OAuth2Filter) parseJWT(token string, discovered *Discovered) (jwt.MapClaims, error) {
	jwtParser := jwt.Parser{
		ValidMethods: []string{
			// Any of the RSA algs supported by jwt-go
			"RS256",
			"RS384",
			"RS512",
		},
		SkipClaimsValidation: true,
	}

	var claims jwt.MapClaims
	_, err := jwtsupport.SanitizeParse(jwtParser.ParseWithClaims(token, &claims, func(t *jwt.Token) (interface{}, error) {
		// Validates key id header.
		if t.Header["kid"] == nil {
			return nil, errors.New("missing kid")
		}

		kid, ok := t.Header["kid"].(string)
		if !ok {
			return nil, errors.New("kid is not a string")
		}

		// Get RSA public key
		return discovered.JSONWebKeySet.GetKey(kid)
	}))
	if err != nil {
		return nil, err
	}
	return claims, nil
}

func (rs *OAuth2Filter) validateScope(actual rfc6749common.Scope) error {
	desired := make(rfc6749common.Scope, len(rs.Arguments.Scopes))
	for _, s := range rs.Arguments.Scopes {
		desired[s] = struct{}{}
	}
	var missing []string
	for scopeValue := range desired {
		if scopeValue == "offline_access" {
			continue
		}
		if _, ok := actual[scopeValue]; !ok {
			missing = append(missing, scopeValue)
		}
	}
	switch len(missing) {
	case 0:
		return nil
	case 1:
		return errors.Errorf("missing required scope value: %q", missing[0])
	default:
		return errors.Errorf("missing required scope values: %q", missing)
	}
}

func (rs *OAuth2Filter) validateJWT(claims jwt.MapClaims, discovered *Discovered, logger types.Logger) error {
	// Validate 'exp', 'iat', and 'nbf' claims.
	if err := claims.Valid(); err != nil {
		return err
	}

	// Validate 'aud' claim.
	//if !claims.VerifyAudience(rs.Spec.Audience, false) {
	//	return errors.Errorf("token has wrong audience: token=%#v expected=%q", claims["aud"], rs.Spec.Audience)
	//}

	// Validate 'iss' claim.
	if !claims.VerifyIssuer(discovered.Issuer, false) {
		return errors.Errorf("token has wrong issuer: token=%#v expected=%q", claims["iss"], discovered.Issuer)
	}

	// Validate 'scopes' claim (draft standard).
	// https://www.iana.org/assignments/jwt/jwt.xhtml
	// https://tools.ietf.org/html/draft-ietf-oauth-token-exchange-16#section-4.2
	switch scopeClaim := claims["scope"].(type) {
	case nil:
		logger.Debugf("No scope to verify")
	case string: // proposed standard; most Authorization Servers do this
		if err := rs.validateScope(rfc6749common.ParseScope(scopeClaim)); err != nil {
			return errors.Wrap(err, "token has wrong scope")
		}
	case []interface{}: // UAA does this
		actual := make(rfc6749common.Scope, len(scopeClaim))
		for _, scopeValue := range scopeClaim {
			switch scopeValue := scopeValue.(type) {
			case string:
				actual[scopeValue] = struct{}{}
			default:
				logger.Warningf("Unexpected scope[n] type: %T", scopeValue)
			}
		}
		if err := rs.validateScope(actual); err != nil {
			return errors.Wrap(err, "token has wrong scope")
		}
	default:
		logger.Warningf("Unexpected scope type: %T", scopeClaim)
	}

	return nil
}
