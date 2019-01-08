package middleware

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/datawire/ambassador-oauth/cmd/ambassador-oauth/config"
	"github.com/datawire/ambassador-oauth/cmd/ambassador-oauth/controller"
	"github.com/datawire/ambassador-oauth/cmd/ambassador-oauth/discovery"
	"github.com/datawire/ambassador-oauth/handler"
	"github.com/datawire/ambassador-oauth/util"
	"github.com/dgrijalva/jwt-go"
	"github.com/sirupsen/logrus"
)

// JWTCheck middleware validates signed tokens when present in the request.
type JWTCheck struct {
	Logger    *logrus.Entry
	Discovery *discovery.Discovery
	Config    *config.Config
	IssuerURL string
}

func (j *JWTCheck) ServeHTTP(w http.ResponseWriter, r *http.Request, next http.HandlerFunc) {
	// Check if Bearer token or cookie exists, otherwise call the next.
	token := j.getToken(r)
	if token == "" {
		j.Logger.Debugf("token not present in the request")

		next(w, r)
		return
	}

	tenant := controller.GetTenantFromContext(r.Context())
	if tenant == nil {
		j.Logger.Errorf("App context cannot be nil")
		util.ToJSONResponse(w, http.StatusUnauthorized, &util.Error{Message: "unauthorized"})
		return
	}

	rule := controller.GetRuleFromContext(r.Context())
	if rule == nil {
		j.Logger.Errorf("Rule context cannot be nil")
		util.ToJSONResponse(w, http.StatusUnauthorized, &util.Error{Message: "unauthorized"})
		return
	}

	// JWT validation is performed by doing the cheap operations first.
	_, err := jwt.Parse(token, func(t *jwt.Token) (interface{}, error) {
		// Validates key id header.
		if t.Header["kid"] == nil {
			return "", errors.New("missing kid")
		}

		// Get RSA certificate.
		cert, err := j.Discovery.GetPemCert(t.Header["kid"].(string))
		if err != nil {
			return "", err
		}

		// Get map of claims.
		claims, ok := t.Claims.(jwt.MapClaims)
		if !ok {
			return "", errors.New("failed to extract claims")
		}

		// Verifies 'aud' claim.
		if !claims.VerifyAudience(tenant.Audience, false) {
			return "", errors.New("invalid audience")
		}

		// Verifies 'iss' claim.
		if !claims.VerifyIssuer(j.Config.IssuerURL, false) {
			return "", errors.New("invalid issuer")
		}

		// Validates time based claims "exp, iat, nbf".
		if err := t.Claims.Valid(); err != nil {
			return "", err
		}

		// Validate scopes.
		if claims["scope"] != nil {
			for _, s := range strings.Split(claims["scope"].(string), " ") {
				j.Logger.Debugf("verifying scope %s", s)
				if !rule.MatchScope(s) {
					return "", fmt.Errorf("scope %v is not in the policy", s)
				}
			}
		}

		// Validate method for last since it's the most expensive operation.
		if _, ok := t.Method.(*jwt.SigningMethodRSA); !ok {
			return "", errors.New("unexpected signing method")
		}

		return jwt.ParseRSAPublicKeyFromPEM([]byte(cert))
	})

	// Since the application received an invalid jwt, clean AccessToken cookie if any and
	// call the next handler.
	if err != nil {
		j.Logger.Debug(err)

		next(w, r)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (j *JWTCheck) getToken(r *http.Request) string {
	cookie, _ := r.Cookie(handler.AccessTokenCookie)
	if cookie != nil {
		return cookie.Value
	}

	j.Logger.Debugf("request has no %s cookie", handler.AccessTokenCookie)

	bearer := strings.Split(r.Header.Get("Authorization"), " ")
	if len(bearer) != 2 && strings.ToLower(bearer[0]) != "bearer" {
		j.Logger.Debug("authorization header is not a bearer token")
		return ""
	}

	return bearer[1]
}
