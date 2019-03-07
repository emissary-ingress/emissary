package app

import (
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/pkg/errors"

	crd "github.com/datawire/apro/apis/getambassador.io/v1beta2"
	"github.com/datawire/apro/cmd/amb-sidecar/filters/app/jwthandler"
	"github.com/datawire/apro/cmd/amb-sidecar/filters/app/middleware"
	"github.com/datawire/apro/cmd/amb-sidecar/filters/app/oauth2handler"
	"github.com/datawire/apro/cmd/amb-sidecar/filters/app/secret"
	"github.com/datawire/apro/cmd/amb-sidecar/filters/controller"
	"github.com/datawire/apro/cmd/amb-sidecar/types"
	"github.com/datawire/apro/lib/mapstructure"
	"github.com/datawire/apro/lib/util"
)

type FilterMux struct {
	Controller   *controller.Controller
	DefaultRule  *crd.Rule
	OAuth2Secret *secret.Secret
	Logger       types.Logger
}

type responseWriter struct {
	status        int
	header        http.Header
	headerWritten bool
	body          strings.Builder
}

func (rw *responseWriter) Header() http.Header {
	return rw.header
}

func (rw *responseWriter) Write(b []byte) (int, error) {
	if !rw.headerWritten {
		rw.WriteHeader(http.StatusOK)
	}
	return rw.body.Write(b)
}

func (rw *responseWriter) WriteHeader(statusCode int) {
	if rw.headerWritten {
		return
	}
	rw.headerWritten = true
	rw.status = statusCode
}

func (rw *responseWriter) reset() {
	rw.body.Reset()
	rw.headerWritten = false
}

func (rw *responseWriter) Status() int {
	return rw.status
}

func (src *responseWriter) writeToResponseWriter(dst http.ResponseWriter) {
	for k, s := range src.header {
		dst.Header()[k] = s
	}
	dst.WriteHeader(src.status)
	io.WriteString(dst, src.body.String())
}

func (c *FilterMux) ServeHTTP(hw http.ResponseWriter, hr *http.Request) {
	// middleware.Logger needs a ResponseWriter implementing
	// .Status()
	w := &responseWriter{
		status: http.StatusOK,
		header: http.Header{},
	}
	mw := &middleware.Logger{Logger: c.Logger}
	mw.ServeHTTP(w, hr, c.serveHTTP)
	w.writeToResponseWriter(hw)
}

func (c *FilterMux) serveHTTP(w http.ResponseWriter, r *http.Request) {
	logger := middleware.GetLogger(r.Context())
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

	response := &responseWriter{
		status: http.StatusOK,
		header: http.Header{},
	}
	for _, filterRef := range rule.Filters {
		filterQName := filterRef.Name + "." + filterRef.Namespace
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
			if err := mapstructure.Convert(filterRef.Arguments, &_handler.FilterArguments); err != nil {
				logger.Errorln("invalid filter.argument:", err)
				util.ToJSONResponse(w, http.StatusInternalServerError, &util.Error{Message: "unauthorized"})
			}
			handler = _handler
		case crd.FilterPlugin:
			handler = filterT.Handler
		case crd.FilterJWT:
			handler = &jwthandler.JWTHandler{
				Filter: filterT,
			}
		default:
			panic(errors.Errorf("unexpected filter type %T", filter))
		}

		response.reset()
		request := new(http.Request)
		*request = *r
		for k, s := range response.header {
			request.Header[k] = s
		}
		handler.ServeHTTP(response, request)
		if response.status != http.StatusOK {
			break
		}
	}
	response.writeToResponseWriter(w)
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
