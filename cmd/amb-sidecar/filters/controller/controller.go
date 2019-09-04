package controller

import (
	"context"
	"strings"
	"sync/atomic"

	"github.com/pkg/errors"

	k8sClientCoreV1 "k8s.io/client-go/kubernetes/typed/core/v1"

	"github.com/datawire/teleproxy/pkg/k8s"

	crd "github.com/datawire/apro/apis/getambassador.io/v1beta2"
	"github.com/datawire/apro/cmd/amb-sidecar/filters/app/httpclient"
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
func (c *Controller) Watch(ctx context.Context, kubeinfo *k8s.KubeInfo) error {
	c.Rules.Store([]crd.Rule{})
	c.Filters.Store(map[string]interface{}{})

	restconfig, err := kubeinfo.GetRestConfig()
	if err != nil {
		return err
	}
	coreClient, err := k8sClientCoreV1.NewForConfig(restconfig)
	if err != nil {
		return err
	}

	w := k8s.NewClient(kubeinfo).Watcher()

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

			if countTrue(spec.OAuth2 != nil, spec.Plugin != nil, spec.JWT != nil, spec.External != nil, spec.Internal != nil) != 1 {
				c.Logger.Errorf("filter resource: must specify exactly 1 of: %v", []string{
					"OAuth2",
					"Plugin",
					"JWT",
					"External",
					"Internal",
				})
				continue
			}

			switch {
			case spec.OAuth2 != nil:
				if err = spec.OAuth2.Validate(mw.Namespace(), coreClient); err != nil {
					c.Logger.Errorln(errors.Wrap(err, "filter resource"))
					continue
				}

				c.Logger.Infof("loading filter domain=%s, client_id=%s", spec.OAuth2.Domain(), spec.OAuth2.ClientID)
				filters[mw.QName()] = *spec.OAuth2
			case spec.Plugin != nil:
				if err = spec.Plugin.Validate(); err != nil {
					c.Logger.Errorln(errors.Wrap(err, "filter resource"))
					continue
				}

				c.Logger.Infof("loading filter plugin=%s", spec.Plugin.Name)
				filters[mw.QName()] = *spec.Plugin
			case spec.JWT != nil:
				if err = spec.JWT.Validate(); err != nil {
					c.Logger.Errorln(errors.Wrap(err, "filter resource"))
					continue
				}

				c.Logger.Infoln("loading filter jwt")
				filters[mw.QName()] = *spec.JWT
			case spec.External != nil:
				if err = spec.External.Validate(); err != nil {
					c.Logger.Errorln(errors.Wrap(err, "filter resource"))
					continue
				}

				c.Logger.Infoln("loading filter external=%s", spec.External.AuthService)
			case spec.Internal != nil:
				c.Logger.Infoln("loading filter internal")
				filters[mw.QName()] = *spec.Internal
			default:
				panic("should not happen")
			}
		}

		if len(filters) == 0 {
			c.Logger.Error("0 filters configured")
		}

		c.Filters.Store(filters)

		// I (lukeshu) measured Auth0 as using ~3.5KiB.
		//
		//    $ curl -is https://ambassador-oauth-e2e.auth0.com/.well-known/openid-configuration https://ambassador-oauth-e2e.auth0.com/.well-known/openid-configuration|wc --bytes
		//    3536
		//
		// Let's go ahead and give each IDP 8KiB, to make sure
		// they have room to breathe.
		httpclient.SetHTTPCacheMaxSize(int64(len(filters)) * 8 * 1024)
	})

	w.Watch("filterpolicies", func(w *k8s.Watcher) {
		rules := make([]crd.Rule, 1)

		// callback is always default.
		rules = append(rules, crd.Rule{
			Host: "*",
			Path: "/callback",
		})
		for _, p := range w.List("filterpolicies") {
			logger := c.Logger.WithField("FILTERPOLICY", p.QName())

			var spec crd.FilterPolicySpec
			err := mapstructure.Convert(p.Spec(), &spec)
			if err != nil {
				logger.Errorln(errors.Wrap(err, "malformed filter policy resource spec"))
				continue
			}
			if c.Config.AmbassadorSingleNamespace && p.Namespace() != c.Config.AmbassadorNamespace {
				continue
			}
			if !spec.AmbassadorID.Matches(c.Config.AmbassadorID) {
				continue
			}

			for _, rule := range spec.Rules {
				if err := rule.Validate(p.Namespace()); err != nil {
					logger.Errorln(errors.Wrap(err, "filter policy resource rule"))
					continue
				}

				filterStrs := make([]string, 0, len(rule.Filters))
				for _, filterRef := range rule.Filters {
					filterStrs = append(filterStrs, filterRef.Name+"."+filterRef.Namespace)
				}
				logger.Infof("loading rule host=%s, path=%s, filters=[%s]",
					rule.Host, rule.Path, strings.Join(filterStrs, ", "))

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
	return nil
}
