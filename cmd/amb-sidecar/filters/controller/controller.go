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

type FilterInfo = crd.FilterInfo

func (c *Controller) storeFilters(filters map[string]FilterInfo) {
	c.filters.Store(filters)
}

func (c *Controller) LoadFilters() map[string]FilterInfo {
	untyped := c.filters.Load()
	if untyped == nil {
		return nil
	}
	typed, ok := untyped.(map[string]FilterInfo)
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

func processFilterSpec(
	filter k8s.Resource,
	cfg types.Config,
	coreClient *k8sClientCoreV1.CoreV1Client,
	haveRedis bool,
) FilterInfo {
	if cfg.AmbassadorSingleNamespace && filter.Namespace() != cfg.AmbassadorNamespace {
		return FilterInfo{Err: &NotThisAmbassadorError{
			Message: fmt.Sprintf("AMBASSADOR_SINGLE_NAMESPACE: .metadata.namespace=%q != AMBASSADOR_NAMESPACE=%q", filter.Namespace(), cfg.AmbassadorNamespace),
		}}
	}
	var spec crd.FilterSpec
	if err := mapstructure.Convert(filter.Spec(), &spec); err != nil {
		return FilterInfo{Err: errors.Wrap(err, "malformed filter resource spec")}
	}
	if !spec.AmbassadorID.Matches(cfg.AmbassadorID) {
		return FilterInfo{Err: &NotThisAmbassadorError{
			Message: fmt.Sprintf("AMBASSADOR_ID: .spec.ambassador_id=%v not contains AMBASSADOR_ID=%q", spec.AmbassadorID, cfg.AmbassadorID),
		}}
	}
	return spec.Validate(filter.Namespace(), coreClient, haveRedis)
}

// Watch monitor changes in k8s cluster and updates rules
func (c *Controller) Watch(
	ctx context.Context,
	kubeinfo *k8s.KubeInfo,
	haveRedis bool,
) error {
	c.storeRules([]crd.Rule{})
	c.storeFilters(map[string]FilterInfo{})

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
		filters := map[string]FilterInfo{}
		for _, mw := range w.List("filters") {
			filterInfo := processFilterSpec(mw, c.Config, coreClient, haveRedis)
			if filterInfo.Err != nil {
				if _, notThisAmbassador := filterInfo.Err.(*NotThisAmbassadorError); notThisAmbassador {
					c.Logger.Debugf("ignoring filter resource %q: %v", mw.QName(), filterInfo.Err)
				} else {
					c.Logger.Errorf("error in filter resource %q: %v", mw.QName(), filterInfo.Err)
				}
			} else {
				c.Logger.Infof("loaded filter resource %q: %v", mw.QName(), filterInfo.Desc)
			}
			filters[mw.QName()] = filterInfo
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
