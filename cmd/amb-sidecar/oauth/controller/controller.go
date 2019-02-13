package controller

import (
	"context"
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
	Logger      types.Logger
	Config      types.Config
	Rules       atomic.Value
	Middlewares atomic.Value
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
	c.Middlewares.Store(map[string]interface{}{})

	w := k8s.NewClient(nil).Watcher()

	w.Watch("middlewares", func(w *k8s.Watcher) {
		middlewares := map[string]interface{}{}
		for _, mw := range w.List("middlewares") {
			var spec crd.MiddlewareSpec
			err := mapstructure.Convert(mw.Spec(), &spec)
			if err != nil {
				c.Logger.Errorln(errors.Wrap(err, "malformed middleware resource spec"))
				continue
			}
			if c.Config.AmbassadorSingleNamespace && mw.Namespace() != c.Config.AmbassadorNamespace {
				continue
			}
			if !spec.AmbassadorID.Matches(c.Config.AmbassadorID) {
				continue
			}

			if countTrue(spec.OAuth2 != nil) != 1 {
				c.Logger.Errorf("middleware resource: must specify exactly 1 of: %v",
					[]string{"OAuth2"})
				continue
			}

			switch {
			case spec.OAuth2 != nil:
				if err = spec.OAuth2.Validate(); err != nil {
					c.Logger.Errorln(errors.Wrap(err, "middleware resource"))
					continue
				}

				c.Logger.Infof("loading middleware domain=%s, client_id=%s", spec.OAuth2.Domain(), spec.OAuth2.ClientID)
				middlewares[mw.QName()] = *spec.OAuth2
			default:
				panic("should not happen")
			}
		}

		if len(middlewares) == 0 {
			c.Logger.Error("0 middlewares configured")
		}

		c.Middlewares.Store(middlewares)
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

				if rule.Middleware.Namespace == "" {
					rule.Middleware.Namespace = p.Namespace()
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
