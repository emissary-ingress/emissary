package app

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"time"

	jwt "github.com/dgrijalva/jwt-go"
	uuid "github.com/satori/go.uuid"

	"github.com/datawire/apro/cmd/amb-sidecar/oauth/app/client"
	"github.com/datawire/apro/cmd/amb-sidecar/oauth/app/discovery"
	"github.com/datawire/apro/cmd/amb-sidecar/oauth/app/secret"
	"github.com/datawire/apro/cmd/amb-sidecar/oauth/controller"
	"github.com/datawire/apro/cmd/amb-sidecar/types"
	"github.com/datawire/apro/lib/util"
)

const (
	// AccessTokenCookie cookie's name
	AccessTokenCookie = "access_token"
	// Code is the default grant used in for this handler.
	Code = "authorization_code"
)

// Handler is the last handler in the chain of the authorization
// server.  If the request Path is "/callback", it validates IDP
// requests and handles code exchange flow.
type Handler struct {
	Config    types.Config
	Logger    types.Logger
	Ctrl      *controller.Controller
	Secret    *secret.Secret
	Discovery *discovery.Discovery
	Rest      *client.Rest
}

// ServeHTTP is a handler function that inspects the request by looking for the presence of
// a token and for any insvalid scope. If these validations pass, an authorization
// header is set in a 200 OK response.
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.URL.Path {
	case "/callback":
		h.Logger.Debug("request received")
		if err := r.URL.Query().Get("error"); err != "" {
			util.ToJSONResponse(w, http.StatusUnauthorized, &util.Error{Message: "unauthorized"})
			return
		}
	default:
	}

	tenant := controller.GetTenantFromContext(r.Context())
	if tenant == nil {
		h.Logger.Errorf("authorization handler: app request context cannot be nil")
		util.ToJSONResponse(w, http.StatusUnauthorized, &util.Error{Message: "unauthorized"})
		return
	}

	switch r.URL.Path {
	case "/callback":
		code := r.URL.Query().Get("code")
		if code == "" {
			h.Logger.Error("check code failed")
			util.ToJSONResponse(w, http.StatusUnauthorized, &util.Error{Message: "unauthorized"})
			return
		}

		rURL, err := h.checkState(r)
		if err != nil {
			h.Logger.Errorf("check state failed: %v", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		var res *client.AuthorizationResponse
		res, err = h.Rest.Authorize(&client.AuthorizationRequest{
			GrantType:    Code,
			ClientID:     tenant.ClientID,
			Code:         code,
			RedirectURL:  tenant.CallbackURL,
			ClientSecret: tenant.Secret,
		})
		if err != nil {
			h.Logger.Errorf("authorization request failed: %v", err)
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		h.Logger.Debug("setting authorization cookie")
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
		h.Logger.Debugf("redirecting user-agent to: %s", rURL)
		http.Redirect(w, r, rURL, http.StatusTemporaryRedirect)
	default:
		rule := controller.GetRuleFromContext(r.Context())
		if rule == nil {
			h.Logger.Errorf("Rule context cannot be nil")
			util.ToJSONResponse(w, http.StatusUnauthorized, &util.Error{Message: "unauthorized"})
			return
		}

		redirect, _ := h.Discovery.AuthorizationEndpoint.Parse("?" + url.Values{
			"audience":      {tenant.Audience},
			"response_type": {"code"},
			"redirect_uri":  {tenant.CallbackURL},
			"client_id":     {tenant.ClientID},
			"state":         {h.signState(r)},
			"scope":         {rule.Scope},
		}.Encode())

		h.Logger.Tracef("redirecting to the authorization endpoint: %s", redirect)
		http.Redirect(w, r, redirect.String(), http.StatusSeeOther)
	}
}

func (h *Handler) signState(r *http.Request) string {
	t := jwt.New(jwt.SigningMethodRS256)
	t.Claims = jwt.MapClaims{
		"exp":          time.Now().Add(h.Config.StateTTL).Unix(), // time when the token will expire (10 minutes from now)
		"jti":          uuid.Must(uuid.NewV4(), nil).String(),    // a unique identifier for the token
		"iat":          time.Now().Unix(),                        // when the token was issued/created (now)
		"nbf":          0,                                        // time before which the token is not yet valid (2 minutes ago)
		"redirect_url": util.OriginalURL(r).String(),             // original request url
	}

	k, err := t.SignedString(h.Secret.GetPrivateKey())
	if err != nil {
		h.Logger.Errorf("failed to sign state: %v", err)
	}

	return k
}

func (c *Handler) checkState(r *http.Request) (string, error) {
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
