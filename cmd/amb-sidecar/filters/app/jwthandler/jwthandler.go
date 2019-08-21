package jwthandler

import (
	"net/http"
	"strings"
	"time"

	jwt "github.com/dgrijalva/jwt-go"
	"github.com/pkg/errors"

	crd "github.com/datawire/apro/apis/getambassador.io/v1beta2"
	"github.com/datawire/apro/cmd/amb-sidecar/filters/app/httpclient"
	"github.com/datawire/apro/cmd/amb-sidecar/filters/app/middleware"
	"github.com/datawire/apro/lib/jwks"
	"github.com/datawire/apro/lib/jwtsupport"
	"github.com/datawire/apro/lib/util"
)

func inArray(needle string, haystack []string) bool {
	for _, straw := range haystack {
		if needle == straw {
			return true
		}
	}
	return false
}

type JWTHandler struct {
	Filter crd.FilterJWT
}

func (h *JWTHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	logger := middleware.GetLogger(r.Context())
	httpClient := httpclient.NewHTTPClient(logger, 0, h.Filter.InsecureTLS)

	token := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")

	if err := validateToken(token, h.Filter, httpClient); err != nil {
		logger.Infoln(err)
		util.ToJSONResponse(w, http.StatusUnauthorized, &util.Error{Message: err.Error()})
	} else {
		w.WriteHeader(http.StatusOK)
	}
}

func validateToken(token string, filter crd.FilterJWT, httpClient *http.Client) error {
	jwtParser := jwt.Parser{ValidMethods: filter.ValidAlgorithms}

	var claims jwt.MapClaims
	_, err := jwtsupport.SanitizeParse(jwtParser.ParseWithClaims(token, &claims, func(t *jwt.Token) (interface{}, error) {
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
		return err
	}

	now := time.Now().Unix()

	if filter.RequireAudience || filter.Audience != "" {
		if !claims.VerifyAudience(filter.Audience, filter.RequireAudience) {
			return errors.Errorf("Token has wrong audience: token=%#v expected=%q", claims["aud"], filter.Audience)
		}
	}

	if filter.RequireIssuer || filter.Issuer != "" {
		if !claims.VerifyIssuer(filter.Issuer, filter.RequireIssuer) {
			return errors.Errorf("Token has wrong issuer: token=%#v expected=%q", claims["iss"], filter.Issuer)
		}
	}

	if !claims.VerifyExpiresAt(now, filter.RequireExpiresAt) {
		return errors.New("Token is expired")
	}

	if !claims.VerifyIssuedAt(now, filter.RequireIssuedAt) {
		return errors.New("Token used before issued")
	}

	if !claims.VerifyNotBefore(now, filter.RequireNotBefore) {
		return errors.New("Token is not valid yet")
	}

	return nil
}
