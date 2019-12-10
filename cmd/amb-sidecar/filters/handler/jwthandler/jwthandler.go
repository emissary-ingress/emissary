package jwthandler

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/datawire/ambassador/pkg/dlog"
	"github.com/dgrijalva/jwt-go"
	"github.com/pkg/errors"

	crd "github.com/datawire/apro/apis/getambassador.io/v1beta2"
	"github.com/datawire/apro/cmd/amb-sidecar/filters/handler/httpclient"
	"github.com/datawire/apro/cmd/amb-sidecar/filters/handler/middleware"
	"github.com/datawire/apro/lib/filterapi"
	"github.com/datawire/apro/lib/filterapi/filterutil"
	"github.com/datawire/apro/lib/jwks"
	"github.com/datawire/apro/lib/jwtsupport"
	"github.com/datawire/apro/resourceserver/rfc6749"
	"github.com/datawire/apro/resourceserver/rfc6750"
)

func inArray(needle string, haystack []string) bool {
	for _, straw := range haystack {
		if needle == straw {
			return true
		}
	}
	return false
}

type JWTFilter struct {
	Spec      crd.FilterJWT
	Arguments crd.FilterJWTArguments
}

type errorWithValidationError struct {
	error
	ValidationError interface{}
}

func (h *JWTFilter) Filter(ctx context.Context, r *filterapi.FilterRequest) (filterapi.FilterResponse, error) {
	logger := dlog.GetLogger(ctx)
	httpClient := httpclient.NewHTTPClient(logger, 0, h.Spec.InsecureTLS, h.Spec.RenegotiateTLS)

	var tokenParsed *jwt.Token
	var hackKeepOldTemplatesWorking error
	validator := &rfc6750.AuthorizationValidator{
		Realm: h.Spec.ErrorResponse.Realm,
		RequiredScope: func() rfc6749.Scope {
			desired := make(rfc6749.Scope, len(h.Arguments.Scope))
			for _, scopeValue := range h.Arguments.Scope {
				if scopeValue == "offline_access" {
					continue
				}
				desired[scopeValue] = struct{}{}
			}
			return desired
		}(),
		TokenValidationFunc: func(tokenString string) (scope rfc6749.Scope, reasonInvalid, serverError error) {
			var claims jwt.MapClaims
			tokenParsed, claims, reasonInvalid, serverError = validateToken(tokenString, h.Spec, httpClient)
			hackKeepOldTemplatesWorking = reasonInvalid
			scope = GetScope(logger, claims)
			return
		},
	}
	err := validator.ValidateAuthorization(&http.Request{
		Header: filterutil.GetHeader(r),
	})
	if err != nil {
		switch err := err.(type) {
		case *rfc6750.AuthorizationError:
			if hackKeepOldTemplatesWorking == nil {
				hackKeepOldTemplatesWorking = err
			}
			if _, isValidationError := hackKeepOldTemplatesWorking.(*jwtsupport.JWTGoError); !isValidationError {
				hackKeepOldTemplatesWorking = errorWithValidationError{
					error: hackKeepOldTemplatesWorking,
				}
			}
			ret := middleware.NewTemplatedErrorResponse(&h.Spec.ErrorResponse, ctx, err.HTTPStatusCode, hackKeepOldTemplatesWorking, map[string]interface{}{
				"httpRequestHeader": filterutil.GetHeader(r),
			})
			if _, overridden := ret.Header[http.CanonicalHeaderKey("WWW-Authenticate")]; !overridden {
				ret.Header.Set("WWW-Authenticate", err.Challenge.String())
			}
			return ret, nil
		default:
			return middleware.NewErrorResponse(ctx, http.StatusInternalServerError, err, nil), nil
		}
	}

	ret := &filterapi.HTTPRequestModification{}
	for _, hf := range h.Spec.InjectRequestHeaders {
		data := map[string]interface{}{
			// "token" is intentionally similar to a
			// *jwt.Token, but unwrapped a bit, since I
			// don't want the jwt-go implementation to be
			// part of our user-facing interface.
			//
			//"token": token,
			"token": map[string]interface{}{
				"Raw":       tokenParsed.Raw,
				"Header":    tokenParsed.Header,
				"Claims":    (map[string]interface{})(*(tokenParsed.Claims.(*jwt.MapClaims))),
				"Signature": tokenParsed.Signature,
			},
			"httpRequestHeader": filterutil.GetHeader(r),
		}
		value := new(strings.Builder)
		if err := hf.Template.Execute(value, data); err != nil {
			return middleware.NewErrorResponse(ctx, http.StatusInternalServerError,
				errors.Wrapf(err, "computing header field %q", hf.Name),
				map[string]interface{}{
					"httpRequestHeader": filterutil.GetHeader(r),
				},
			), nil
		}
		ret.Header = append(ret.Header, &filterapi.HTTPHeaderReplaceValue{
			Key:   hf.Name,
			Value: value.String(),
		})
	}

	return ret, nil
}

func stringInArray(needle string, haystack []string) bool {
	for _, straw := range haystack {
		if straw == needle {
			return true
		}
	}
	return false
}

func validateToken(signedString string, filter crd.FilterJWT, httpClient *http.Client) (token *jwt.Token, claims jwt.MapClaims, reasonInvalid, serverError error) {
	// Get the key
	keys, err := jwks.FetchJWKS(httpClient, filter.JSONWebKeySetURI.String())
	if err != nil {
		return nil, nil, nil, err
	}

	jwtParser := jwt.Parser{ValidMethods: filter.ValidAlgorithms}
	token, err = jwtsupport.SanitizeParse(jwtParser.ParseWithClaims(signedString, &claims, func(t *jwt.Token) (interface{}, error) {
		if t.Method == jwt.SigningMethodNone && inArray("none", filter.ValidAlgorithms) {
			return jwt.UnsafeAllowNoneSignatureType, nil
		}

		// Validate key ID header.
		if t.Header["kid"] == nil {
			return nil, errors.New("missing kid")
		}
		kid, ok := t.Header["kid"].(string)
		if !ok {
			return nil, errors.New("kid is not a string")
		}

		return keys.GetKey(kid)
	}))
	if err != nil {
		return nil, nil, err, nil
	}

	now := time.Now().Unix()

	if filter.RequireAudience || filter.Audience != "" {
		audienceClaim := GetAudience(claims)
		if filter.RequireAudience || len(audienceClaim) > 0 {
			if !stringInArray(filter.Audience, audienceClaim) {
				return nil, nil, errors.Errorf("Token has wrong audience: token=%#v expected=%q", audienceClaim, filter.Audience), nil
			}
		}
	}

	if filter.RequireIssuer || filter.Issuer != "" {
		if !claims.VerifyIssuer(filter.Issuer, filter.RequireIssuer) {
			return nil, nil, errors.Errorf("Token has wrong issuer: token=%#v expected=%q", claims["iss"], filter.Issuer), nil
		}
	}

	if !claims.VerifyExpiresAt(now, filter.RequireExpiresAt) {
		return nil, nil, errors.New("Token is expired"), nil
	}

	if !claims.VerifyIssuedAt(now, filter.RequireIssuedAt) {
		return nil, nil, errors.New("Token used before issued"), nil
	}

	if !claims.VerifyNotBefore(now, filter.RequireNotBefore) {
		return nil, nil, errors.New("Token is not valid yet"), nil
	}

	return token, claims, nil, nil
}
