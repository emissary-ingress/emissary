package middleware

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/datawire/ambassador-oauth/cmd/ambassador-oauth/client"

	jwt "github.com/dgrijalva/jwt-go"

	"github.com/datawire/ambassador-oauth/cmd/ambassador-oauth/config"
	"github.com/datawire/ambassador-oauth/cmd/ambassador-oauth/secret"
	"github.com/datawire/ambassador-oauth/util"
	"github.com/sirupsen/logrus"
)

const (
	// AccessTokenCookie cookie's name
	AccessTokenCookie = "access_token"
	// Code is the default grant used in for this handler.
	Code = "authorization_code"
	// CallbackPath is the default callback path URL
	CallbackPath = "/callback"
)

// Callback ...
type Callback struct {
	Logger *logrus.Logger
	Config *config.Config
	Secret *secret.Secret
	Rest   *client.Rest
}

func (c *Callback) ServeHTTP(rw http.ResponseWriter, r *http.Request, next http.HandlerFunc) {
	if r.URL.Path == CallbackPath {
		c.Logger.Debug("received callback request")
		if err := r.URL.Query().Get("error"); err != "" {
			util.ToJSONResponse(rw, http.StatusUnauthorized, &util.Error{Message: "unauthorized"})
			return
		}

		redirectPath, err := c.checkState(r)
		if err != nil {
			c.Logger.Errorf("check state failed: %v", err)
			rw.WriteHeader(http.StatusBadRequest)
			return
		}

		code := r.URL.Query().Get("code")
		if code == "" {
			c.Logger.Error("check code failed")
			util.ToJSONResponse(rw, http.StatusUnauthorized, &util.Error{Message: "unauthorized"})
			return
		}

		rq := &client.AuthorizationRequest{
			GrantType:    Code,
			ClientID:     c.Config.ClientID,
			Code:         code,
			RedirectURL:  c.Config.CallbackURL,
			ClientSecret: c.Config.ClientSecret,
		}

		var rs *client.AuthorizationResponse
		rs, err = c.Rest.POSTAuthorization(rq)
		if err != nil {
			c.Logger.Errorf("authorization request failed:", err)
			rw.WriteHeader(http.StatusBadRequest)
			return
		}

		c.Logger.Debugf("setting access_token cookie from authorization %s token", rs.TokenType)
		http.SetCookie(rw, &http.Cookie{
			Name:     AccessTokenCookie,
			Value:    rs.AccessToken,
			HttpOnly: true,
			Expires:  time.Now().Add(time.Duration(rs.ExpiresIn) * time.Second),
		})

		// If the user-agent request was a POST or PUT, 307 will preserve the body
		// and just follow the location header.
		// https://tools.ietf.org/html/rfc7231#section-6.4.7
		http.Redirect(rw, r, redirectPath, http.StatusTemporaryRedirect)
		c.Logger.Debugf("redirecting to path: %s", redirectPath)
		return
	}

	next(rw, r)
}

func (c *Callback) checkState(r *http.Request) (string, error) {
	state := r.URL.Query().Get("state")
	if state == "" {
		return "", errors.New("empty state param")
	}

	token, err := jwt.Parse(state, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			return "", fmt.Errorf("unexpected signing method %v", token.Header["path"])
		}
		return c.Secret.GetPublicKey(), nil
	})

	if err != nil {
		return "", err
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !(ok && token.Valid) {
		return "", errors.New("state token validation failed")
	}

	return claims["path"].(string), nil
}
