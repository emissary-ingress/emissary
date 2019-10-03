package oauth2handler

import (
	"context"
	"net/http"

	jwt "github.com/dgrijalva/jwt-go"
	"github.com/pkg/errors"

	rfc6749client "github.com/datawire/liboauth2/client/rfc6749"
	rfc6750resourceserver "github.com/datawire/liboauth2/resourceserver/rfc6750"

	"github.com/datawire/apro/cmd/amb-sidecar/filters/handler/middleware"
	"github.com/datawire/apro/cmd/amb-sidecar/types"
	"github.com/datawire/apro/lib/filterapi"
	"github.com/datawire/apro/lib/filterapi/filterutil"
	"github.com/datawire/apro/lib/jwtsupport"
)

// filterResourceServer implements the OAuth Resource Server part of the Filter.
func (c *OAuth2Filter) filterResourceServer(ctx context.Context, logger types.Logger, httpClient *http.Client, discovered *Discovered, request *filterapi.FilterRequest) filterapi.FilterResponse {
	token := rfc6750resourceserver.GetFromHeader(filterutil.GetHeader(request))
	if err := c.validateAccessToken(token, discovered, httpClient, logger); err != nil {
		return middleware.NewErrorResponse(ctx, http.StatusBadRequest, err, nil)
	}
	return nil
}

func (j *OAuth2Filter) validateAccessToken(token string, discovered *Discovered, httpClient *http.Client, logger types.Logger) error {
	switch j.Spec.AccessTokenValidation {
	case "auto":
		claims, err := j.parseJWT(token, discovered)
		if err == nil {
			return j.validateJWT(claims, discovered, logger)
		}
		logger.Debugln("rejecting JWT validation; falling back to UserInfo Endpoint validation:", err)
		fallthrough
	case "userinfo":
		return j.validateAccessTokenUserinfo(token, discovered, httpClient, logger)
	case "jwt":
		claims, err := j.parseJWT(token, discovered)
		if err != nil {
			return err
		}
		return j.validateJWT(claims, discovered, logger)
	}
	panic("not reached")
}

func (j *OAuth2Filter) parseJWT(token string, discovered *Discovered) (jwt.MapClaims, error) {
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

func (j *OAuth2Filter) validateScope(actual rfc6749client.Scope) error {
	desired := make(rfc6749client.Scope, len(j.Arguments.Scopes))
	for _, s := range j.Arguments.Scopes {
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

func (j *OAuth2Filter) validateJWT(claims jwt.MapClaims, discovered *Discovered, logger types.Logger) error {
	// Validate 'exp', 'iat', and 'nbf' claims.
	if err := claims.Valid(); err != nil {
		return err
	}

	// Validate 'aud' claim.
	//if !claims.VerifyAudience(j.Spec.Audience, false) {
	//	return errors.Errorf("token has wrong audience: token=%#v expected=%q", claims["aud"], j.Spec.Audience)
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
		if err := j.validateScope(rfc6749client.ParseScope(scopeClaim)); err != nil {
			return errors.Wrap(err, "token has wrong scope")
		}
	case []interface{}: // UAA does this
		actual := make(rfc6749client.Scope, len(scopeClaim))
		for _, scopeValue := range scopeClaim {
			switch scopeValue := scopeValue.(type) {
			case string:
				actual[scopeValue] = struct{}{}
			default:
				logger.Warningf("Unexpected scope[n] type: %T", scopeValue)
			}
		}
		if err := j.validateScope(actual); err != nil {
			return errors.Wrap(err, "token has wrong scope")
		}
	default:
		logger.Warningf("Unexpected scope type: %T", scopeClaim)
	}

	return nil
}
