package jwthandler

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/pkg/errors"

	"github.com/datawire/apro/resourceserver/rfc6750"

	crd "github.com/datawire/apro/apis/getambassador.io/v1beta2"
	"github.com/datawire/apro/cmd/amb-sidecar/filters/handler/httpclient"
	"github.com/datawire/apro/cmd/amb-sidecar/filters/handler/middleware"
	"github.com/datawire/apro/lib/filterapi"
	"github.com/datawire/apro/lib/filterapi/filterutil"
	"github.com/datawire/apro/lib/jwks"
	"github.com/datawire/apro/lib/jwtsupport"
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
	Spec crd.FilterJWT
}

func (h *JWTFilter) Filter(ctx context.Context, r *filterapi.FilterRequest) (filterapi.FilterResponse, error) {
	logger := middleware.GetLogger(ctx)
	httpClient := httpclient.NewHTTPClient(logger, 0, h.Spec.InsecureTLS, h.Spec.RenegotiateTLS)

	tokenString := rfc6750.GetFromHeader(filterutil.GetHeader(r))

	token, err := validateToken(tokenString, h.Spec, httpClient)
	if err != nil {
		if h.Spec.ErrorResponse.BodyTemplate != nil {
			return middleware.TemplatedErrorResponse(ctx, http.StatusUnauthorized, err, *h.Spec.ErrorResponse.BodyTemplate, h.Spec.ErrorResponse.ContentType), nil
		} else {
			return middleware.NewErrorResponse(ctx, http.StatusUnauthorized, err, nil), nil
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
				"Raw":       token.Raw,
				"Header":    token.Header,
				"Claims":    (map[string]interface{})(*(token.Claims.(*jwt.MapClaims))),
				"Signature": token.Signature,
			},
		}
		value := new(strings.Builder)
		if err := hf.Template.Execute(value, data); err != nil {
			return middleware.NewErrorResponse(ctx, http.StatusInternalServerError,
				errors.Wrapf(err, "computing header field %q", hf.Name), nil), nil
		}
		ret.Header = append(ret.Header, &filterapi.HTTPHeaderReplaceValue{
			Key:   hf.Name,
			Value: value.String(),
		})
	}

	return ret, nil
}

func validateToken(signedString string, filter crd.FilterJWT, httpClient *http.Client) (*jwt.Token, error) {
	jwtParser := jwt.Parser{ValidMethods: filter.ValidAlgorithms}

	var claims jwt.MapClaims
	token, err := jwtsupport.SanitizeParse(jwtParser.ParseWithClaims(signedString, &claims, func(t *jwt.Token) (interface{}, error) {
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

		// Get the key
		keys, err := jwks.FetchJWKS(httpClient, filter.JSONWebKeySetURI.String())
		if err != nil {
			return nil, err
		}
		return keys.GetKey(kid)
	}))
	if err != nil {
		return nil, err
	}

	now := time.Now().Unix()

	if filter.RequireAudience || filter.Audience != "" {
		if !claims.VerifyAudience(filter.Audience, filter.RequireAudience) {
			return nil, errors.Errorf("Token has wrong audience: token=%#v expected=%q", claims["aud"], filter.Audience)
		}
	}

	if filter.RequireIssuer || filter.Issuer != "" {
		if !claims.VerifyIssuer(filter.Issuer, filter.RequireIssuer) {
			return nil, errors.Errorf("Token has wrong issuer: token=%#v expected=%q", claims["iss"], filter.Issuer)
		}
	}

	if !claims.VerifyExpiresAt(now, filter.RequireExpiresAt) {
		return nil, errors.New("Token is expired")
	}

	if !claims.VerifyIssuedAt(now, filter.RequireIssuedAt) {
		return nil, errors.New("Token used before issued")
	}

	if !claims.VerifyNotBefore(now, filter.RequireNotBefore) {
		return nil, errors.New("Token is not valid yet")
	}

	return token, nil
}
