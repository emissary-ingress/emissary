package middleware

import (
	"context"
	"net/http"

	crd "github.com/datawire/apro/apis/getambassador.io/v1beta1"
	"github.com/datawire/apro/cmd/amb-sidecar/oauth/controller"
	"github.com/datawire/apro/cmd/amb-sidecar/types"
	"github.com/datawire/apro/lib/util"
)

// ControllerCheck looks up the appropriate Tenant and Rule objects
// from the CRD Controller.  The objects are injected in to the
// Request Context.
type ControllerCheck struct {
	Logger      types.Logger
	Ctrl        *controller.Controller
	DefaultRule *crd.Rule
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
