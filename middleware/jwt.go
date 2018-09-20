package middleware

// TODO(gsagula): lots to clean up in this file

import (
	"errors"
	"fmt"
	"net/http"

	jwtmidleware "github.com/auth0/go-jwt-middleware"
	"github.com/datawire/ambassador-oauth/cmd/ambassador-oauth/config"
	"github.com/datawire/ambassador-oauth/cmd/ambassador-oauth/discovery"
	jwt "github.com/dgrijalva/jwt-go"
	"github.com/sirupsen/logrus"
)

// JWT ...
type JWT struct {
	Logger    *logrus.Logger
	Config    *config.Config
	Discovery *discovery.Discovery
	mw        *jwtmidleware.JWTMiddleware
}

// ServeHTTP ..
func (j *JWT) ServeHTTP(rw http.ResponseWriter, r *http.Request, next http.HandlerFunc) {
	if j.mw == nil {
		j.init()
	}

	err := j.mw.CheckJWT(rw, r)
	if err != nil {
		rw.WriteHeader(http.StatusUnauthorized)
		return
	}

	next(rw, r)
}

func (j *JWT) init() {
	j.mw = jwtmidleware.New(jwtmidleware.Options{
		ValidationKeyGetter: func(token *jwt.Token) (interface{}, error) {
			claims := token.Claims.(jwt.MapClaims)
			// Verifies 'aud' claim
			if !claims.VerifyAudience(j.Config.Audience, false) {
				return token, errors.New("invalid audience")
			}

			// Verifies 'iss' claim
			if !claims.VerifyIssuer(fmt.Sprintf("%s://%s/", j.Config.Scheme, j.Config.Domain), false) {
				return token, errors.New("invalid issuer")
			}

			// Validates time based claims "exp, iat, nbf".
			if err := token.Claims.Valid(); err != nil {
				return token, err
			}

			// Validates key id header
			if token.Header["kid"] == nil {
				return token, errors.New("missing kid")
			}

			cert, err := j.Discovery.GetPemCert(token.Header["kid"].(string))
			if err != nil {
				j.Logger.Fatal(err)
			}

			return jwt.ParseRSAPublicKeyFromPEM([]byte(cert))
		},
		SigningMethod:       jwt.SigningMethodRS256,
		CredentialsOptional: true,
	})
}
