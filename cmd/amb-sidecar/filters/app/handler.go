package app

import (
	"net/http"
	"net/url"

	"github.com/pkg/errors"

	crd "github.com/datawire/apro/apis/getambassador.io/v1beta2"
	"github.com/datawire/apro/cmd/amb-sidecar/filters/app/middleware"
	"github.com/datawire/apro/cmd/amb-sidecar/filters/app/oauth2handler"
	"github.com/datawire/apro/cmd/amb-sidecar/filters/app/secret"
	"github.com/datawire/apro/cmd/amb-sidecar/filters/controller"
	"github.com/datawire/apro/lib/mapstructure"
	"github.com/datawire/apro/lib/util"
)

type FilterHandler struct {
	Controller   *controller.Controller
	DefaultRule  *crd.Rule
	OAuth2Secret *secret.Secret
}

func (c *FilterHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	logger := middleware.GetLogger(r)
	originalURL := util.OriginalURL(r)

	var rule *crd.Rule
	var redirectURL *url.URL
	switch originalURL.Path {
	case "/callback":
		redirectURLstr, err := oauth2handler.CheckState(r, c.OAuth2Secret)
		if err != nil {
			logger.Errorf("check state failed: %v", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		redirectURL, err = url.Parse(redirectURLstr)
		if err != nil {
			logger.Errorf("could not parse JWT redirect_url claim: %q: %v", redirectURLstr, err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		rule = findRule(c.Controller, redirectURL.Host, redirectURL.Path)
	default:
		rule = findRule(c.Controller, originalURL.Host, originalURL.Path)
	}
	if rule == nil {
		rule = c.DefaultRule
	}
	if rule.Filter == nil {
		logger.Debugf("%s %s is public", originalURL.Host, originalURL.Path)
		w.WriteHeader(http.StatusOK)
		return
	}
	filterQName := rule.Filter.Name + "." + rule.Filter.Namespace
	logger.Debugf("host=%s, path=%s, filter=%q", rule.Host, rule.Path, filterQName)

	filter := findFilter(c.Controller, filterQName)
	if filter == nil {
		logger.Debugf("could not find not filter: %q", filterQName)
		util.ToJSONResponse(w, http.StatusUnauthorized, &util.Error{Message: "unauthorized"})
		return
	}

	var handler http.Handler
	switch filterT := filter.(type) {
	case crd.FilterOAuth2:
		_handler := &oauth2handler.OAuth2Handler{
			Secret:      c.OAuth2Secret,
			Filter:      filterT,
			OriginalURL: originalURL,
			RedirectURL: redirectURL,
		}
		if err := mapstructure.Convert(rule.Filter.Arguments, &_handler.FilterArguments); err != nil {
			logger.Errorln("invalid filter.argument:", err)
			util.ToJSONResponse(w, http.StatusInternalServerError, &util.Error{Message: "unauthorized"})
		}
		handler = _handler
	case crd.FilterPlugin:
		handler = filterT.Handler
	default:
		panic(errors.Errorf("unexpected filter type %T", filter))
	}
	handler.ServeHTTP(w, r)
}

func findFilter(c *controller.Controller, qname string) interface{} {
	mws := c.Filters.Load()
	if mws != nil {
		filters := mws.(map[string]interface{})
		filter, ok := filters[qname]
		if ok {
			return filter
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
