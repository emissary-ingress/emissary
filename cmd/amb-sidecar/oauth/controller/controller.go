package controller

import (
	"context"
	"net/http"
	"plugin"
	"strings"
	"sync/atomic"

	"github.com/pkg/errors"

	"github.com/datawire/teleproxy/pkg/k8s"

	crd "github.com/datawire/apro/apis/getambassador.io/v1beta1"
	"github.com/datawire/apro/cmd/amb-sidecar/types"
	"github.com/datawire/apro/lib/mapstructure"
)

// Controller is monitors changes in app configuration and policy custom resources.
type Controller struct {
	Logger  types.Logger
	Config  types.Config
	Rules   atomic.Value
	Filters atomic.Value
}

func countTrue(args ...bool) int {
	n := 0
	for _, arg := range args {
		if arg {
			n++
		}
	}
	return n
}

// Watch monitor changes in k8s cluster and updates rules
func (c *Controller) Watch(ctx context.Context) {
	c.Rules.Store([]crd.Rule{})
	c.Filters.Store(map[string]interface{}{})

	w := k8s.NewClient(nil).Watcher()

	w.Watch("filters", func(w *k8s.Watcher) {
		filters := map[string]interface{}{}
		for _, mw := range w.List("filters") {
			var spec crd.FilterSpec
			err := mapstructure.Convert(mw.Spec(), &spec)
			if err != nil {
				c.Logger.Errorln(errors.Wrap(err, "malformed filter resource spec"))
				continue
			}
			if c.Config.AmbassadorSingleNamespace && mw.Namespace() != c.Config.AmbassadorNamespace {
				continue
			}
			if !spec.AmbassadorID.Matches(c.Config.AmbassadorID) {
				continue
			}

			if countTrue(spec.OAuth2 != nil, spec.Plugin != nil) != 1 {
				c.Logger.Errorf("filter resource: must specify exactly 1 of: %v",
					[]string{"OAuth2", "Plugin"})
				continue
			}

			switch {
			case spec.OAuth2 != nil:
				if err = spec.OAuth2.Validate(); err != nil {
					c.Logger.Errorln(errors.Wrap(err, "filter resource"))
					continue
				}

				c.Logger.Infof("loading filter domain=%s, client_id=%s", spec.OAuth2.Domain(), spec.OAuth2.ClientID)
				filters[mw.QName()] = *spec.OAuth2
			case spec.Plugin != nil:
				if strings.Contains(spec.Plugin.Name, "/") {
					c.Logger.Errorf("filter resource: invalid Plugin.name: contains a /: %q", spec.Plugin.Name)
					continue
				}
				p, err := plugin.Open("/etc/ambassador-plugins/" + spec.Plugin.Name + ".so")
				if err != nil {
					c.Logger.Errorln("filter resource: could not open plugin file:", err)
					continue
				}
				f, err := p.Lookup("PluginMain")
				if err != nil {
					c.Logger.Errorln("filter resource: invalid plugin file:", err)
					continue
				}
				h, ok := f.(func(http.ResponseWriter, *http.Request))
				if !ok {
					c.Logger.Errorln("filter resource: invalid plugin file: PluginMain has the wrong type")
					continue
				}
				spec.Plugin.Handler = http.HandlerFunc(h)

				c.Logger.Infof("loading filter plugin=%s", spec.Plugin.Name)
				filters[mw.QName()] = *spec.Plugin
			default:
				panic("should not happen")
			}
		}

		if len(filters) == 0 {
			c.Logger.Error("0 filters configured")
		}

		c.Filters.Store(filters)
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
			if c.Config.AmbassadorSingleNamespace && p.Namespace() != c.Config.AmbassadorNamespace {
				continue
			}
			if !spec.AmbassadorID.Matches(c.Config.AmbassadorID) {
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

				if rule.Filter.Namespace == "" {
					rule.Filter.Namespace = p.Namespace()
				}

				rules = append(rules, rule)
			}
		}

		c.Rules.Store(rules)
	})

	go func() {
		<-ctx.Done()
		w.Stop()
	}()

	w.Wait()
}
