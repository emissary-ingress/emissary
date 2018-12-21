package handler

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/datawire/ambassador-oauth/cmd/ambassador-oauth/client"
	"github.com/datawire/ambassador-oauth/cmd/ambassador-oauth/controller"
	"github.com/datawire/ambassador-oauth/cmd/ambassador-oauth/secret"
	"github.com/datawire/ambassador-oauth/util"

	"github.com/dgrijalva/jwt-go"

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

// Callback validates IDP requests and handles code exchange flow.
type Callback struct {
	Logger *logrus.Entry
	Secret *secret.Secret
	Ctrl   *controller.Controller
	Rest   *client.Rest
}

// Check inspects if the request contains code and signed states...
func (c *Callback) Check(w http.ResponseWriter, r *http.Request) {
	c.Logger.Debug("request received")
	if err := r.URL.Query().Get("error"); err != "" {
		util.ToJSONResponse(w, http.StatusUnauthorized, &util.Error{Message: "unauthorized"})
		return
	}

	tenant := controller.GetTenantFromContext(r.Context())
	if tenant == nil {
		c.Logger.Errorf("app request context cannot be nil")
		util.ToJSONResponse(w, http.StatusUnauthorized, &util.Error{Message: "unauthorized"})
		return
	}

	code := r.URL.Query().Get("code")
	if code == "" {
		c.Logger.Error("check code failed")
		util.ToJSONResponse(w, http.StatusUnauthorized, &util.Error{Message: "unauthorized"})
		return
	}

	rURL, err := c.checkState(r)
	if err != nil {
		c.Logger.Errorf("check state failed: %v", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	var res *client.AuthorizationResponse
	res, err = c.Rest.Authorize(&client.AuthorizationRequest{
		GrantType:    Code,
		ClientID:     tenant.ClientID,
		Code:         code,
		RedirectURL:  tenant.CallbackURL,
		ClientSecret: tenant.Secret,
	})
	if err != nil {
		c.Logger.Errorf("authorization request failed: %v", err)
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	c.Logger.Debug("setting authorization cookie")
	http.SetCookie(w, &http.Cookie{
		Name:     AccessTokenCookie,
		Value:    res.AccessToken,
		HttpOnly: true,
		Secure:   tenant.TLS,
		Expires:  time.Now().Add(time.Duration(res.ExpiresIn) * time.Second),
	})

	// If the user-agent request was a POST or PUT, 307 will preserve the body
	// and just follow the location header.
	// https://tools.ietf.org/html/rfc7231#section-6.4.7
	c.Logger.Debugf("redirecting user-agent to: %s", rURL)
	http.Redirect(w, r, rURL, http.StatusTemporaryRedirect)
}

func (c *Callback) checkState(r *http.Request) (string, error) {
	state := r.URL.Query().Get("state")
	if state == "" {
		return "", errors.New("empty state param")
	}

	token, err := jwt.Parse(state, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodRSA); !ok {
			return "", fmt.Errorf("unexpected signing method %v", t.Header["redirect_url"])
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

	return claims["redirect_url"].(string), nil
}
