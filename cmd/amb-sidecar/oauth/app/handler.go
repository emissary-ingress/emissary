package app

import (
	"net/http"
	"net/url"

	"github.com/pkg/errors"

	crd "github.com/datawire/apro/apis/getambassador.io/v1beta1"
	"github.com/datawire/apro/cmd/amb-sidecar/oauth/app/oauth2handler"
	"github.com/datawire/apro/cmd/amb-sidecar/oauth/app/secret"
	"github.com/datawire/apro/cmd/amb-sidecar/oauth/controller"
	"github.com/datawire/apro/cmd/amb-sidecar/types"
	"github.com/datawire/apro/lib/util"
)

type MiddlewareHandler struct {
	Logger       types.Logger
	Controller   *controller.Controller
	DefaultRule  *crd.Rule
	OAuth2Secret *secret.Secret
}

func (c *MiddlewareHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	originalURL := util.OriginalURL(r)

	var rule *crd.Rule
	var redirectURL *url.URL
	switch originalURL.Path {
	case "/callback":
		redirectURLstr, err := oauth2handler.CheckState(r, c.OAuth2Secret)
		if err != nil {
			c.Logger.Errorf("check state failed: %v", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		redirectURL, err = url.Parse(redirectURLstr)
		if err != nil {
			c.Logger.Errorf("could not parse JWT redirect_url claim: %q: %v", redirectURLstr, err)
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
	middlewareQName := rule.Middleware.Name + "." + rule.Middleware.Namespace
	c.Logger.Debugf("host=%s, path=%s, public=%v, middleware=%q", rule.Host, rule.Path, rule.Public, middlewareQName)
	if rule.Public {
		c.Logger.Debugf("%s %s is public", originalURL.Host, originalURL.Path)
		w.WriteHeader(http.StatusOK)
		return
	}

	middleware := findMiddleware(c.Controller, middlewareQName)
	if middleware == nil {
		c.Logger.Debugf("could not find not middleware: %q", middlewareQName)
		util.ToJSONResponse(w, http.StatusUnauthorized, &util.Error{Message: "unauthorized"})
		return
	}

	var handler http.Handler
	switch middlewareT := middleware.(type) {
	case crd.MiddlewareOAuth2:
		handler = &oauth2handler.OAuth2Handler{
			Logger:      c.Logger,
			Secret:      c.OAuth2Secret,
			Rule:        *rule,
			Middleware:  middlewareT,
			OriginalURL: originalURL,
			RedirectURL: redirectURL,
		}
	default:
		panic(errors.Errorf("unexpected middleware type %T", middleware))
	}
	handler.ServeHTTP(w, r)
}

func findMiddleware(c *controller.Controller, qname string) interface{} {
	mws := c.Middlewares.Load()
	if mws != nil {
		middlewares := mws.(map[string]interface{})
		middleware, ok := middlewares[qname]
		if ok {
			return middleware
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
