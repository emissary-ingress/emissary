package controller

import (
	"context"
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
	Logger *logrus.Entry
	Config *config.Config
	Rules  atomic.Value
	Apps   atomic.Value
}

const (
	// RuleCTXKey is passed to the the request handler as a context key.
	RuleCTXKey = util.HTTPContextKey("rule")

	// TenantCTXKey is passed to the the request handler as a context key.
	TenantCTXKey = util.HTTPContextKey("tenant")

	// OfflineAccess is a default scope and commonly used scopes, e.g. "openid".
	OfflineAccess = "offline_access"
)

// Watch monitor changes in k8s cluster and updates rules
func (c *Controller) Watch() {
	c.Rules.Store(make([]Rule, 0))
	w := k8s.NewClient(nil).Watcher()

	// TODO(gsagula): DRY here and below.
	w.Watch("tenants", func(w *k8s.Watcher) {
		apps := make([]Tenant, 0)
		for _, p := range w.List("tenants") {
			spec, err := decode(p.QName(), p.Spec())
			if err != nil {
				c.Logger.Errorf("malformed tenant resource spec: %v", spec)
				continue
			}
			for _, r := range spec.Tenants {
				t := Tenant{}
				err := ms.Decode(r, &t)
				if err != nil {
					c.Logger.Errorf("decode tenant failed: %v", err)
				} else {
					c.Logger.Infof("loading tenant: %s: %s", t.Domain, t.ClientID)
					apps = append(apps, t)
				}
			}
		}

		c.Apps.Store(apps)
	})

	w.Watch("policies", func(w *k8s.Watcher) {
		rules := make([]Rule, 0)
		for _, p := range w.List("policies") {
			spec, err := decode(p.QName(), p.Spec())
			if err != nil {
				c.Logger.Errorf("malformed rule resource spec: %v", spec)
				continue
			}
			for _, r := range spec.Rules {
				rule := Rule{}
				err := ms.Decode(r, &rule)

				if err != nil {
					c.Logger.Errorf("decode rule failed: %v", err)
				} else {
					c.Logger.Infof("loading rule host=%s, path=%s, public=%v, scopes=%s",
						rule.Host, rule.Path, rule.Public, rule.Scopes)

					rule.ScopeMap = make(map[string]bool)
					rule.ScopeMap[OfflineAccess] = true
					scopes := strings.Split(rule.Scopes, " ")
					for _, s := range scopes {
						rule.ScopeMap[s] = true
					}

					rules = append(rules, rule)
				}
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
	Host     string
	Path     string
	Public   bool
	Scopes   string
	ScopeMap map[string]bool
}

// Tenant defines a single application object.
type Tenant struct {
	CallbackURL string
	Domain      string
	Audience    string
	ClientID    string
	Secret      string
	Scopes      string
}

// Match return true if rules matches the supplied hostname and path.
func (r Rule) Match(host, path string) bool {
	return match(r.Host, host) && match(r.Path, path)
}

// MatchHost return true if rules matches the supplied hostname.
func (r Rule) MatchHost(host string) bool {
	return match(r.Host, host)
}

// MatchPath return true if rules matches the supplied path.
func (r Rule) MatchPath(path string) bool {
	return match(r.Path, path)
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
	apps := c.Apps.Load()
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
