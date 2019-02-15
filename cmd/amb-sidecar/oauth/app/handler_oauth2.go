package app

import (
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
)

// OAuth2Handler looks up the appropriate Tenant and Rule objects from
// the CRD Controller, and validates the signed JWT tokens when
// present in the request.  If the request Path is "/callback", it
// validates IDP requests and handles code exchange flow.
type OAuth2Handler struct {
	Config      types.Config
	Logger      types.Logger
	Ctrl        *controller.Controller
	DefaultRule *crd.Rule
	IssuerURL   string
	Secret      *secret.Secret
}

func (c *OAuth2Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	originalURL := util.OriginalURL(r)

	var rule *crd.Rule
	var tenant *crd.TenantObject
	var redirectURL *url.URL
	switch originalURL.Path {
	case "/callback":
		redirectURLstr, err := c.checkState(r)
		if err != nil {
			c.Logger.Errorf("check state failed: %v", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		redirectURL, err = url.Parse(redirectURLstr)
		if err != nil {
			c.Logger.Errorf("could not parse JWT redirect_url claim: %q: %v", redirectURLstr, err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		rule = findRule(c.Ctrl, redirectURL.Host, redirectURL.Path)
		tenant = findTenant(c.Ctrl, redirectURL.Host)
	default:
		rule = findRule(c.Ctrl, originalURL.Host, originalURL.Path)
		tenant = findTenant(c.Ctrl, originalURL.Host)
	}
	if rule == nil {
		rule = c.DefaultRule
	}
	c.Logger.Debugf("host=%s, path=%s, public=%v", rule.Host, rule.Path, rule.Public)
	if rule.Public {
		c.Logger.Debugf("%s %s is public", originalURL.Host, originalURL.Path)
		w.WriteHeader(http.StatusOK)
		return
	}
	if tenant == nil {
		c.Logger.Debugf("not a registered domain: %s", originalURL.Host)
		util.ToJSONResponse(w, http.StatusUnauthorized, &util.Error{Message: "unauthorized"})
		return
	}

	middleware := &crd.MiddlewareOAuth2{
		RawAuthorizationURL: c.Config.AuthProviderURL.String(),
		RawClientURL:        tenant.TenantURL.String(),
		RawStateTTL:         c.Config.StateTTL.String(),
		Audience:            tenant.Audience,
		ClientID:            tenant.ClientID,
		Secret:              tenant.Secret,
	}
	if err := middleware.Validate(); err != nil {
		c.Logger.Debugf("create middleware: %v", err)
		util.ToJSONResponse(w, http.StatusUnauthorized, &util.Error{Message: "unauthorized"})
		return
	}

	disco, err := discovery.New(*middleware, c.Logger)
	if err != nil {
		c.Logger.Debugf("create discovery: %v", err)
		util.ToJSONResponse(w, http.StatusUnauthorized, &util.Error{Message: "unauthorized"})
	}

	token := c.getToken(r)
	var tokenErr error
	if token == "" {
		tokenErr = errors.New("token not present in the request")
	} else {
		tokenErr = c.validateToken(token, middleware, rule, disco)
	}
	if tokenErr == nil {
		w.Header().Set("Authorization", fmt.Sprintf("Bearer %s", token))
		w.WriteHeader(http.StatusOK)
		return
	}
	c.Logger.Debug(tokenErr)

	switch originalURL.Path {
	case "/callback":
		if err := r.URL.Query().Get("error"); err != "" {
			util.ToJSONResponse(w, http.StatusUnauthorized, &util.Error{Message: "unauthorized"})
			return
		}

		code := r.URL.Query().Get("code")
		if code == "" {
			c.Logger.Error("check code failed")
			util.ToJSONResponse(w, http.StatusUnauthorized, &util.Error{Message: "unauthorized"})
			return
		}

		var res *client.AuthorizationResponse
		res, err = client.NewRestClient(disco.AuthorizationEndpoint, disco.TokenEndpoint).Authorize(&client.AuthorizationRequest{
			GrantType:    "authorization_code", // the default grant used in for this handler
			ClientID:     middleware.ClientID,
			Code:         code,
			RedirectURL:  middleware.CallbackURL().String(),
			ClientSecret: middleware.Secret,
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
			Secure:   middleware.TLS(),
			Expires:  time.Now().Add(time.Duration(res.ExpiresIn) * time.Second),
		})

		// If the user-agent request was a POST or PUT, 307 will preserve the body
		// and just follow the location header.
		// https://tools.ietf.org/html/rfc7231#section-6.4.7
		c.Logger.Debugf("redirecting user-agent to: %s", redirectURL)
		http.Redirect(w, r, redirectURL.String(), http.StatusTemporaryRedirect)

	default:
		redirect, _ := disco.AuthorizationEndpoint.Parse("?" + url.Values{
			"audience":      {middleware.Audience},
			"response_type": {"code"},
			"redirect_uri":  {middleware.CallbackURL().String()},
			"client_id":     {middleware.ClientID},
			"state":         {c.signState(r, middleware)},
			"scope":         {rule.Scope},
		}.Encode())

		c.Logger.Tracef("redirecting to the authorization endpoint: %s", redirect)
		http.Redirect(w, r, redirect.String(), http.StatusSeeOther)
	}
}

func findTenant(c *controller.Controller, domain string) *crd.TenantObject {
	apps := c.Tenants.Load()
	if apps != nil {
		for _, app := range apps.([]crd.TenantObject) {
			if app.Domain() == domain {
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

func (j *OAuth2Handler) validateToken(token string, middleware *crd.MiddlewareOAuth2, rule *crd.Rule, disco *discovery.Discovery) error {
	// JWT validation is performed by doing the cheap operations first.
	_, err := jwt.Parse(token, func(t *jwt.Token) (interface{}, error) {
		// Validates key id header.
		if t.Header["kid"] == nil {
			return "", errors.New("missing kid")
		}

		// Get RSA certificate.
		cert, err := disco.GetPemCert(t.Header["kid"].(string))
		if err != nil {
			return "", err
		}

		// Get map of claims.
		claims, ok := t.Claims.(jwt.MapClaims)
		if !ok {
			return "", errors.New("failed to extract claims")
		}

		//fmt.Printf("Expected aud: %s\n", middleware.Audience)
		//fmt.Printf("Expected iss: %s\n", j.IssuerURL)
		//spew.Dump(claims)

		// Verifies 'aud' claim.
		if !claims.VerifyAudience(middleware.Audience, false) {
			return "", fmt.Errorf("invalid audience %s", middleware.Audience)
		}

		// Verifies 'iss' claim.
		if !claims.VerifyIssuer(disco.Issuer, false) {
			return "", fmt.Errorf("invalid issuer: %s", disco.Issuer)
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

func (j *OAuth2Handler) getToken(r *http.Request) string {
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

func (h *OAuth2Handler) signState(r *http.Request, middleware *crd.MiddlewareOAuth2) string {
	t := jwt.New(jwt.SigningMethodRS256)
	t.Claims = jwt.MapClaims{
		"exp":          time.Now().Add(middleware.StateTTL).Unix(), // time when the token will expire (10 minutes from now)
		"jti":          uuid.Must(uuid.NewV4(), nil).String(),      // a unique identifier for the token
		"iat":          time.Now().Unix(),                          // when the token was issued/created (now)
		"nbf":          0,                                          // time before which the token is not yet valid (2 minutes ago)
		"redirect_url": util.OriginalURL(r).String(),               // original request url
	}

	k, err := t.SignedString(h.Secret.GetPrivateKey())
	if err != nil {
		h.Logger.Errorf("failed to sign state: %v", err)
	}

	return k
}

func (c *OAuth2Handler) checkState(r *http.Request) (string, error) {
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
