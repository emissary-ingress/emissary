package controller

import (
	"context"
	"fmt"
	"net"
	"net/url"
	"strings"
	"sync/atomic"

	"github.com/datawire/ambassador-oauth/cmd/ambassador-oauth/config"
	"github.com/datawire/ambassador-oauth/util"
	"github.com/datawire/teleproxy/pkg/k8s"
	"github.com/gobwas/glob"
	"github.com/sirupsen/logrus"

	ms "github.com/mitchellh/mapstructure"
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

	// DefaultScope is normally used for when no rule has matched the request path or host.
	DefaultScope = "offline_access"

	// Callback is the path used to create the tenant callback url.
	Callback = "callback"
)

// Watch monitor changes in k8s cluster and updates rules
func (c *Controller) Watch() {
	c.Rules.Store(make([]Rule, 0))
	w := k8s.NewClient(nil).Watcher()

	w.Watch("tenants", func(w *k8s.Watcher) {
		tenants := make([]Tenant, 0)

		for _, p := range w.List("tenants") {
			spec, err := decode(p.QName(), p.Spec())
			if err != nil {
				c.Logger.Errorf("malformed tenant resource spec")
				continue
			}
			for _, r := range spec.Tenants {
				t := Tenant{}

				err := ms.Decode(r, &t)
				if err != nil {
					c.Logger.Errorf("decode tenant failed: %v", err)
					continue
				}

				u, err := url.Parse(t.TenantURL)
				if err != nil {
					c.Logger.Errorf("parsing tenant url: %v", err)
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
		rules := make([]Rule, 1)

		// callback is always default.
		rules = append(rules, Rule{
			Host: "*",
			Path: "/callback",
		})

		for _, p := range w.List("policies") {
			spec, err := decode(p.QName(), p.Spec())

			if err != nil {
				c.Logger.Error("malformed rule resource spec")
				continue
			}

			for _, r := range spec.Rules {
				rule := Rule{}
				err := ms.Decode(r, &rule)

				if err != nil {
					c.Logger.Errorf("decode rule failed: %v", err)
					continue
				}

				c.Logger.Infof("loading rule host=%s, path=%s, public=%v, scope=%s",
					rule.Host, rule.Path, rule.Public, rule.Scope)

				rule.scopes = make(map[string]bool)
				scopes := strings.Split(rule.Scope, " ")
				for _, s := range scopes {
					rule.scopes[s] = true
				}

				rules = append(rules, rule)
			}
		}

		c.Rules.Store(rules)
	})

	w.Wait()
}

// Spec used by the controller to retrieve the authorization rule and app configuration
// resource definitions.
type Spec struct {
	Rules   []Rule
	Tenants []Tenant
}

// Rule defines authorization rules object.
type Rule struct {
	Host   string
	Path   string
	Public bool
	Scope  string
	scopes map[string]bool
}

// Tenant defines a single application object.
type Tenant struct {
	CallbackURL string
	TenantURL   string
	TLS         bool
	Domain      string
	Audience    string
	ClientID    string
	Secret      string
}

// MatchHTTPHeaders return true if rules matches the supplied hostname and path.
func (r Rule) MatchHTTPHeaders(host, path string) bool {
	return match(r.Host, host) && match(r.Path, path)
}

// MatchScope return true if rule scope.
func (r Rule) MatchScope(scope string) bool {
	return r.Scope == DefaultScope || r.scopes[scope]
}

// GetRuleFromContext is a handy method for retrieving a reference of Rule from an HTTP
// request context.
func GetRuleFromContext(ctx context.Context) *Rule {
	if r := ctx.Value(RuleCTXKey); r != nil {
		return r.(*Rule)
	}
	return nil
}

// GetTenantFromContext is a handy method for retrieving a reference of App from an HTTP
// request context.
func GetTenantFromContext(ctx context.Context) *Tenant {
	if a := ctx.Value(TenantCTXKey); a != nil {
		return a.(*Tenant)
	}
	return nil
}

// FindTenant returns app definition resource by looking up the domain name.
func (c *Controller) FindTenant(domain string) *Tenant {
	apps := c.Tenants.Load()
	if apps != nil {
		for _, app := range apps.([]Tenant) {
			if app.isDomain(domain) {
				return &app
			}
		}
	}

	return nil
}

func match(pattern, input string) bool {
	g, err := glob.Compile(pattern)
	if err != nil {
		return false
	}

	return g.Match(input)
}

func (a *Tenant) isDomain(domain string) bool {
	return strings.Compare(a.Domain, domain) == 0
}

func decode(source string, input interface{}) (*Spec, error) {
	var s Spec
	d, err := ms.NewDecoder(&ms.DecoderConfig{
		ErrorUnused: true,
		Result:      &s,
	})
	if err != nil {
		return nil, err
	}

	err = d.Decode(input)
	if err != nil {
		return nil, err
	}
	return &s, nil
}
