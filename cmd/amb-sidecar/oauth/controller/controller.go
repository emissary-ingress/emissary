package controller

import (
	"context"
	"fmt"
	"net"
	"net/url"
	"strings"
	"sync/atomic"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"github.com/datawire/teleproxy/pkg/k8s"

	crd "github.com/datawire/apro/apis/getambassador.io/v1beta1"
	"github.com/datawire/apro/cmd/amb-sidecar/oauth/config"
	"github.com/datawire/apro/lib/mapstructure"
	"github.com/datawire/apro/lib/util"
)

// Controller is monitors changes in app configuration and policy custom resources.
type Controller struct {
	Logger  *logrus.Entry
	Config  *config.Config
	Rules   atomic.Value
	Tenants atomic.Value
}

const (
	// RuleCTXKey is passed to the the request handler as a context key.
	RuleCTXKey = util.HTTPContextKey("rule")

	// TenantCTXKey is passed to the the request handler as a context key.
	TenantCTXKey = util.HTTPContextKey("tenant")

	// Callback is the path used to create the tenant callback url.
	Callback = "callback"
)

// Watch monitor changes in k8s cluster and updates rules
func (c *Controller) Watch() {
	c.Rules.Store(make([]crd.Rule, 0))
	w := k8s.NewClient(nil).Watcher()

	w.Watch("tenants", func(w *k8s.Watcher) {
		tenants := make([]crd.TenantObject, 0)
		for _, p := range w.List("tenants") {
			var spec crd.TenantSpec
			err := mapstructure.Convert(p.Spec(), &spec)
			if err != nil {
				c.Logger.Errorln(errors.Wrap(err, "malformed tenant resource spec"))
				continue
			}

			for _, t := range spec.Tenants {
				u, err := url.Parse(t.TenantURL)
				if err != nil {
					c.Logger.Errorln(errors.Wrap(err, "parsing tenant url"))
					continue
				}

				if u.Scheme == "" {
					c.Logger.Errorf("tenantUrl needs to be an absolute url: {scheme}://{host}:{port}")
					continue
				}

				t.TLS = u.Scheme == "https"

				_, port, _ := net.SplitHostPort(u.Host)
				if port == "" {
					t.CallbackURL = fmt.Sprintf("%s://%s/%s", u.Scheme, u.Host, Callback)
				} else {
					t.CallbackURL = fmt.Sprintf("%s://%s:%s/%s", u.Scheme, u.Host, port, Callback)
				}

				t.Domain = u.Host

				c.Logger.Infof("loading tenant domain=%s, client_id=%s", t.Domain, t.ClientID)

				tenants = append(tenants, t)
			}
		}

		if len(tenants) == 0 {
			c.Logger.Error("0 tenant apps configured")
		}

		c.Tenants.Store(tenants)
	})

	w.Watch("policies", func(w *k8s.Watcher) {
		rules := make([]crd.Rule, 1)

		// callback is always default.
		rules = append(rules, crd.Rule{
			Host: "*",
			Path: "/callback",
		})

		for _, p := range w.List("policies") {
			var spec crd.PolicySpec
			err := mapstructure.Convert(p.Spec(), &spec)
			if err != nil {
				c.Logger.Errorln(errors.Wrap(err, "malformed policy resource spec"))
				continue
			}

			for _, rule := range spec.Rules {
				c.Logger.Infof("loading rule host=%s, path=%s, public=%v, scope=%s",
					rule.Host, rule.Path, rule.Public, rule.Scope)

				rule.Scopes = make(map[string]bool)
				scopes := strings.Split(rule.Scope, " ")
				for _, s := range scopes {
					rule.Scopes[s] = true
				}

				rules = append(rules, rule)
			}
		}

		c.Rules.Store(rules)
	})

	w.Wait()
}

// GetRuleFromContext is a handy method for retrieving a reference of Rule from an HTTP
// request context.
func GetRuleFromContext(ctx context.Context) *crd.Rule {
	if r := ctx.Value(RuleCTXKey); r != nil {
		return r.(*crd.Rule)
	}
	return nil
}

// GetTenantFromContext is a handy method for retrieving a reference of App from an HTTP
// request context.
func GetTenantFromContext(ctx context.Context) *crd.TenantObject {
	if a := ctx.Value(TenantCTXKey); a != nil {
		return a.(*crd.TenantObject)
	}
	return nil
}

// FindTenant returns app definition resource by looking up the domain name.
func (c *Controller) FindTenant(domain string) *crd.TenantObject {
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
