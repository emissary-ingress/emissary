package resourceserver

import (
	"context"
	"net/http"

	jwt "github.com/dgrijalva/jwt-go"
	"github.com/pkg/errors"

	crd "github.com/datawire/apro/apis/getambassador.io/v1beta2"
	"github.com/datawire/apro/cmd/amb-sidecar/filters/handler/jwthandler"
	"github.com/datawire/apro/cmd/amb-sidecar/filters/handler/middleware"
	"github.com/datawire/apro/cmd/amb-sidecar/filters/handler/oauth2handler/discovery"
	"github.com/datawire/apro/cmd/amb-sidecar/types"
	"github.com/datawire/apro/lib/filterapi"
	"github.com/datawire/apro/lib/filterapi/filterutil"
	"github.com/datawire/apro/lib/jwtsupport"
	"github.com/datawire/apro/resourceserver/rfc6749"
	"github.com/datawire/apro/resourceserver/rfc6750"
)

// OAuth2ResourceServer implements the OAuth Resource Server part of the Filter.
type OAuth2ResourceServer struct {
	QName     string
	Spec      crd.FilterOAuth2
	Arguments crd.FilterOAuth2Arguments
}

// Filter kinda implements filterapi.Filter, but takes a bunch of
// extra arguments.  As things get cleaned up, most of the arguments
// can probably go away (hello "dlog"), but it will always need an
// extra `scope` argument, because there isn't actually a way for the
// Resource Server to know the scope of an arbitrary opaque Access
// Token without writing IDP-specific code.  But the (Client) caller
// has that information, so just accept it as an argument (breaking
// the layering/abstraction).
//
// As a "special" case (i.e. not part of the filterapi.Filter
// semantics), a FilterResponse of "nil" means to send the same
// request to the upstream service (the other half of the Resource
// Server).
func (rs *OAuth2ResourceServer) Filter(ctx context.Context, logger types.Logger, httpClient *http.Client, discovered *discovery.Discovered, request *filterapi.FilterRequest, clientScope rfc6749.Scope) filterapi.FilterResponse {
	validator := &rfc6750.AuthorizationValidator{
		Realm: rs.QName,
		RequiredScope: func() rfc6749.Scope {
			desired := make(rfc6749.Scope, len(rs.Arguments.Scopes))
			for _, scopeValue := range rs.Arguments.Scopes {
				if scopeValue == "offline_access" {
					continue
				}
				desired[scopeValue] = struct{}{}
			}
			return desired
		}(),
		TokenValidationFunc: func(token string) (scope rfc6749.Scope, reasonInvalid, serverErr error) {
			jwtScope, jwtErr, serverErr := rs.validateAccessToken(token, discovered, httpClient, logger)
			if serverErr != nil {
				return nil, nil, serverErr
			}
			if jwtErr != nil {
				return nil, jwtErr, nil
			}
			// We took the scope as an argument from the client, instead of extracting it from the
			// authorization token, because there isn't actually a good portable way to extract it
			// from the authorization token (it's IDP-specific).  We can get away with avoiding
			// IDP-specific behavior because we trust the client.
			switch len(jwtScope) {
			case 0:
				return clientScope, nil, nil
			default:
				// However, if we did extract a scope claim from a JWT authorization token,
				// then go ahead and validate it too.  We use the intersection of the
				// client-reported scope and the jwt-reported scope; effectively validating
				// *both*.
				unionScope := make(rfc6749.Scope)
				for scopeValue := range clientScope {
					if _, both := jwtScope[scopeValue]; both {
						unionScope[scopeValue] = struct{}{}
					}
				}
				return unionScope, nil, nil
			}
		},
	}
	err := validator.ValidateAuthorization(&http.Request{
		Header: filterutil.GetHeader(request),
	})
	if err != nil {
		switch err := err.(type) {
		case *rfc6750.AuthorizationError:
			ret := middleware.NewErrorResponse(ctx, err.HTTPStatusCode, err, nil)
			ret.Header.Set("WWW-Authenticate", err.Challenge.String())
			return ret
		default:
			return middleware.NewErrorResponse(ctx, http.StatusInternalServerError, err, nil)
		}
	}
	// If everything has passed, go ahead and have Envoy proxy to the other half
	// of the Resource Server.
	return nil
}

func (rs *OAuth2ResourceServer) validateAccessToken(token string, discovered *discovery.Discovered, httpClient *http.Client, logger types.Logger) (scope rfc6749.Scope, tokenErr error, serverErr error) {
	switch rs.Spec.AccessTokenValidation {
	case "auto":
		claims, err := rs.parseJWT(token, discovered)
		if err == nil {
			scope, tokenErr = rs.validateJWT(claims, discovered, logger)
			return scope, tokenErr, nil
		}
		logger.Debugln("rejecting JWT validation; falling back to UserInfo Endpoint validation:", err)
		fallthrough
	case "userinfo":
		tokenErr, serverErr = rs.validateAccessTokenUserinfo(token, discovered, httpClient, logger)
		return nil, tokenErr, serverErr
	case "jwt":
		claims, err := rs.parseJWT(token, discovered)
		if err != nil {
			return nil, err, nil
		}
		scope, tokenErr = rs.validateJWT(claims, discovered, logger)
		return scope, tokenErr, nil
	}
	panic("not reached")
}

func (rs *OAuth2ResourceServer) parseJWT(token string, discovered *discovery.Discovered) (jwt.MapClaims, error) {
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

func (rs *OAuth2ResourceServer) validateJWT(claims jwt.MapClaims, discovered *discovery.Discovered, logger types.Logger) (rfc6749.Scope, error) {
	// Validate 'exp', 'iat', and 'nbf' claims.
	if err := claims.Valid(); err != nil {
		return nil, err
	}

	// Validate 'aud' claim.
	//if !claims.VerifyAudience(rs.Spec.Audience, false) {
	//	return errors.Errorf("token has wrong audience: token=%#v expected=%q", claims["aud"], rs.Spec.Audience)
	//}

	// Validate 'iss' claim.
	if !claims.VerifyIssuer(discovered.Issuer, false) {
		return nil, errors.Errorf("token has wrong issuer: token=%#v expected=%q", claims["iss"], discovered.Issuer)
	}

	return jwthandler.GetScope(logger, claims), nil
}
