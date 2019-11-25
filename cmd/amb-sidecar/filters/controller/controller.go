package controller

import (
	"context"
	"fmt"
	"strings"
	"sync/atomic"

	"github.com/pkg/errors"

	k8sClientCoreV1 "k8s.io/client-go/kubernetes/typed/core/v1"

	"github.com/datawire/ambassador/pkg/dlog"
	"github.com/datawire/ambassador/pkg/k8s"
	crd "github.com/datawire/apro/apis/getambassador.io/v1beta2"
	"github.com/datawire/apro/cmd/amb-sidecar/filters/handler/httpclient"
	"github.com/datawire/apro/cmd/amb-sidecar/types"
	"github.com/datawire/apro/lib/mapstructure"
)

// Controller is monitors changes in app configuration and policy custom resources.
type Controller struct {
	Logger  dlog.Logger
	Config  types.Config
	rules   atomic.Value
	filters atomic.Value
}

func (c *Controller) storeRules(rules []crd.Rule) {
	c.rules.Store(rules)
}

func (c *Controller) LoadRules() []crd.Rule {
	untyped := c.rules.Load()
	if untyped == nil {
		return nil
	}
	typed, ok := untyped.([]crd.Rule)
	if !ok {
		return nil
	}
	return typed
}

func (c *Controller) storeFilters(filters map[string]crd.Filter) {
	c.filters.Store(filters)
}

func (c *Controller) LoadFilters() map[string]crd.Filter {
	untyped := c.filters.Load()
	if untyped == nil {
		return nil
	}
	typed, ok := untyped.(map[string]crd.Filter)
	if !ok {
		return nil
	}
	return typed
}

type NotThisAmbassadorError struct {
	Message string
}

func (e *NotThisAmbassadorError) Error() string {
	return e.Message
}

func parseFilter(untypedFilter k8s.Resource, cfg types.Config) (crd.Filter, error) {
	if cfg.AmbassadorSingleNamespace && untypedFilter.Namespace() != cfg.AmbassadorNamespace {
		return crd.Filter{}, &NotThisAmbassadorError{
			Message: fmt.Sprintf("AMBASSADOR_SINGLE_NAMESPACE: .metadata.namespace=%q != AMBASSADOR_NAMESPACE=%q", untypedFilter.Namespace(), cfg.AmbassadorNamespace),
		}
	}
	var filter crd.Filter
	if err := mapstructure.Convert(untypedFilter, &filter); err != nil {
		return crd.Filter{}, errors.Wrap(err, "malformed filter resource spec")
	}
	if !filter.Spec.AmbassadorID.Matches(cfg.AmbassadorID) {
		return crd.Filter{}, &NotThisAmbassadorError{
			Message: fmt.Sprintf("AMBASSADOR_ID: .spec.ambassador_id=%v does not contain AMBASSADOR_ID=%q", filter.Spec.AmbassadorID, cfg.AmbassadorID),
		}
	}
	return filter, nil
}

// Watch monitor changes in k8s cluster and updates rules
func (c *Controller) Watch(
	ctx context.Context,
	kubeinfo *k8s.KubeInfo,
	haveRedis bool,
) error {
	c.storeRules([]crd.Rule{})
	c.storeFilters(map[string]crd.Filter{})

	restconfig, err := kubeinfo.GetRestConfig()
	if err != nil {
		return err
	}
	coreClient, err := k8sClientCoreV1.NewForConfig(restconfig)
	if err != nil {
		return err
	}

	client, err := k8s.NewClient(kubeinfo)
	if err != nil {
		// this is non fatal (mostly just to facilitate local dev); don't `return err`
		c.Logger.Errorln("not watching Filter or FilterPolicy resources:", errors.Wrap(err, "k8s.NewClient"))
		return nil
	}
	w := client.Watcher()

	w.Watch("filters", func(w *k8s.Watcher) {
		filters := map[string]crd.Filter{}
		for _, untypedFilter := range w.List("filters") {
			filter, err := parseFilter(untypedFilter, c.Config)
			if err != nil {
				if _, notThisAmbassador := err.(*NotThisAmbassadorError); notThisAmbassador {
					c.Logger.Debugf("ignoring Filter resource %q: %v", untypedFilter.QName(), err)
				} else {
					c.Logger.Errorf("malformed Filter resource %q: %v", untypedFilter.QName(), err)
				}
				continue
			}
			if err := filter.Validate(coreClient, haveRedis); err != nil {
				c.Logger.Errorf("error in Filter resource %q: %v", untypedFilter.QName(), err)
			}
			c.Logger.Infof("loaded filter resource %q: %v", untypedFilter.QName(), filter.Desc)
			filters[untypedFilter.QName()] = filter
		}

		if len(filters) == 0 {
			c.Logger.Error("0 filters configured")
		}

		c.storeFilters(filters)

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
		var rules []crd.Rule

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

		c.storeRules(rules)
	})

	go func() {
		<-ctx.Done()
		w.Stop()
	}()

	w.Wait()
	return nil
}
