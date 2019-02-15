package app

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	jwt "github.com/dgrijalva/jwt-go"
	uuid "github.com/satori/go.uuid"

	crd "github.com/datawire/apro/apis/getambassador.io/v1beta1"
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

// ControllerCheck looks up the appropriate Tenant and Rule objects
// from the CRD Controller, and validates the signed JWT tokens when
// present in the request.  The Tenant and Rule objects are injected
// in to the Request Context.
type ControllerCheck struct {
	Logger      types.Logger
	Ctrl        *controller.Controller
	DefaultRule *crd.Rule
	Discovery   *discovery.Discovery
	Config      types.Config
	IssuerURL   string
}

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

func (c *ControllerCheck) ServeHTTP(w http.ResponseWriter, r *http.Request, next http.HandlerFunc) {
	originalURL := util.OriginalURL(r)

	tenant := findTenant(c.Ctrl, originalURL.Hostname())
	if tenant == nil {
		c.Logger.Debugf("not a registered domain: %s", originalURL.Hostname())
		util.ToJSONResponse(w, http.StatusUnauthorized, &util.Error{Message: "unauthorized"})
		return
	}

	rule := findRule(c.Ctrl, originalURL.Host, originalURL.Path)
	if rule == nil {
		rule = c.DefaultRule
	}
	c.Logger.Debugf("host=%s, path=%s, public=%v", rule.Host, rule.Path, rule.Public)
	if rule.Public {
		c.Logger.Debugf("%s %s is public", originalURL.Host, originalURL.Path)
		w.WriteHeader(http.StatusOK)
		return
	}

	ctx := r.Context()
	ctx = context.WithValue(ctx, controller.TenantCTXKey, tenant)
	ctx = context.WithValue(ctx, controller.RuleCTXKey, rule)

	token := c.getToken(r)
	var tokenErr error
	if token == "" {
		tokenErr = errors.New("token not present in the request")
	} else {
		tokenErr = c.validateToken(token, tenant, rule)
	}
	if tokenErr == nil {
		w.Header().Set("Authorization", fmt.Sprintf("Bearer %s", token))
		w.WriteHeader(http.StatusOK)
		return
	}
	c.Logger.Debug(tokenErr)

	next(w, r.WithContext(ctx))
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

func findTenant(c *controller.Controller, domain string) *crd.TenantObject {
	apps := c.Tenants.Load()
	if apps != nil {
		for _, app := range apps.([]crd.TenantObject) {
			if app.Domain == domain {
				return &app
			}
		}
	}

	return nil
}

func findRule(c *controller.Controller, host, path string) *crd.Rule {
	rules := c.Rules.Load()
	if rules != nil {
		for _, rule := range rules.([]crd.Rule) {
			if rule.MatchHTTPHeaders(host, path) {
				return &rule
			}
		}
	}

	return nil
}

func (j *ControllerCheck) validateToken(token string, tenant *crd.TenantObject, rule *crd.Rule) error {
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

		//fmt.Printf("Expected aud: %s\n", tenant.Audience)
		//fmt.Printf("Expected iss: %s\n", j.IssuerURL)
		//spew.Dump(claims)

		// Verifies 'aud' claim.
		if !claims.VerifyAudience(tenant.Audience, false) {
			return "", fmt.Errorf("invalid audience %s", tenant.Audience)
		}

		// Verifies 'iss' claim.
		if !claims.VerifyIssuer(j.IssuerURL, false) {
			return "", fmt.Errorf("invalid issuer %s", j.IssuerURL)
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
	return err
}

func (j *ControllerCheck) getToken(r *http.Request) string {
	cookie, _ := r.Cookie(AccessTokenCookie)
	if cookie != nil {
		return cookie.Value
	}

	j.Logger.Debugf("request has no %s cookie", AccessTokenCookie)

	bearer := strings.Split(r.Header.Get("Authorization"), " ")
	if len(bearer) != 2 && strings.ToLower(bearer[0]) != "bearer" {
		j.Logger.Debug("authorization header is not a bearer token")
		return ""
	}

	return bearer[1]
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
