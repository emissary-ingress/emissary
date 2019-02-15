package app

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	jwt "github.com/dgrijalva/jwt-go"

	crd "github.com/datawire/apro/apis/getambassador.io/v1beta1"
	"github.com/datawire/apro/cmd/amb-sidecar/oauth/app/discovery"
	"github.com/datawire/apro/cmd/amb-sidecar/oauth/controller"
	"github.com/datawire/apro/cmd/amb-sidecar/types"
	"github.com/datawire/apro/lib/util"
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
	IssuerURL string
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
