package middleware

// TODO(gsagula): lots to clean up in this file

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	jwtm "github.com/auth0/go-jwt-middleware"
	"github.com/datawire/ambassador-oauth/cmd/ambassador-oauth/config"
	jwt "github.com/dgrijalva/jwt-go"
	"github.com/sirupsen/logrus"
)

const (
	// WellKnownURLFmt ..
	WellKnownURLFmt = "https://%s/.well-known/jwks.json"
)

// Jwt ...
type Jwt struct {
	Logger *logrus.Logger
	Config *config.Config
}

// Middleware ..
func (j *Jwt) Middleware() *jwtm.JWTMiddleware {
	m := jwtm.New(jwtm.Options{
		ValidationKeyGetter: func(token *jwt.Token) (interface{}, error) {
			c := token.Claims.(jwt.MapClaims)
			// Verify 'aud' claim
			if !c.VerifyAudience(j.Config.Audience, false) {
				return token, errors.New("Invalid audience")
			}

			// Verify 'iss' claim
			if !c.VerifyIssuer(fmt.Sprintf("https://%s/", j.Config.Domain), false) {
				return token, errors.New("Invalid issuer")
			}

			cert, err := j.getPemCert(token)
			if err != nil {
				panic(err.Error())
			}

			result, _ := jwt.ParseRSAPublicKeyFromPEM([]byte(cert))
			return result, nil
		},
		SigningMethod:       jwt.SigningMethodRS256,
		CredentialsOptional: true,
	})

	return m
}

// JSONWebKeys TODO(gsagula): comment
type JSONWebKeys struct {
	Kty string   `json:"kty"`
	Kid string   `json:"kid"`
	Use string   `json:"use"`
	N   string   `json:"n"`
	E   string   `json:"e"`
	X5c []string `json:"x5c"`
}

// Jwks TODO(gsagula): comment
type Jwks struct {
	Keys []JSONWebKeys `json:"keys"`
}

var pemCert = EmptyString

func (j *Jwt) getPemCert(token *jwt.Token) (string, error) {
	var err error
	if pemCert != EmptyString {
		return pemCert, nil
	}
	pemCert, err := j.getPemCertUncached(token)
	return pemCert, err
}

func (j *Jwt) getPemCertUncached(token *jwt.Token) (string, error) {
	cert := EmptyString
	resp, err := http.Get(fmt.Sprintf(WellKnownURLFmt, j.Config.Domain))
	if err != nil {
		return cert, err
	}
	defer resp.Body.Close()

	var jwks = Jwks{}
	err = json.NewDecoder(resp.Body).Decode(&jwks)

	if err != nil {
		return cert, err
	}

	for k := range jwks.Keys {
		if token.Header["kid"] == jwks.Keys[k].Kid {
			cert = "-----BEGIN CERTIFICATE-----\n" + jwks.Keys[k].X5c[0] + "\n-----END CERTIFICATE-----"
		}
	}

	if cert == EmptyString {
		err := errors.New("Unable to find appropriate key")
		return cert, err
	}

	return cert, nil
}
