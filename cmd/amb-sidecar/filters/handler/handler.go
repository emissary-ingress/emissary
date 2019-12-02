package handler

import (
	"context"
	"crypto/rsa"
	"net/http"
	"net/url"
	"strings"
	"time"

	jwt "github.com/dgrijalva/jwt-go"
	"github.com/mediocregopher/radix.v2/pool"
	"github.com/pkg/errors"

	crd "github.com/datawire/apro/apis/getambassador.io/v1beta2"
	"github.com/datawire/apro/cmd/amb-sidecar/filters/controller"
	"github.com/datawire/apro/cmd/amb-sidecar/filters/handler/externalhandler"
	"github.com/datawire/apro/cmd/amb-sidecar/filters/handler/internalhandler"
	"github.com/datawire/apro/cmd/amb-sidecar/filters/handler/jwthandler"
	"github.com/datawire/apro/cmd/amb-sidecar/filters/handler/middleware"
	"github.com/datawire/apro/cmd/amb-sidecar/filters/handler/oauth2handler"
	"github.com/datawire/apro/cmd/amb-sidecar/types"
	"github.com/datawire/apro/lib/filterapi"
	"github.com/datawire/apro/lib/filterapi/filterutil"
	"github.com/datawire/apro/lib/jwtsupport"
	"github.com/datawire/apro/lib/mapstructure"
	"github.com/datawire/apro/lib/util"
)

type FilterMux struct {
	Controller  *controller.Controller
	DefaultRule *crd.Rule
	PrivateKey  *rsa.PrivateKey
	PublicKey   *rsa.PublicKey
	Logger      types.Logger
	RedisPool   *pool.Pool
}

func logResponse(logger types.Logger, ret filterapi.FilterResponse, took time.Duration) {
	switch _ret := ret.(type) {
	case *filterapi.HTTPResponse:
		if _ret == nil {
			logger.Errorf("[gRPC] %T : unexpected nil (%v)", _ret, took)
		} else {
			if loc := _ret.Header.Get("Location"); loc != "" {
				logger.Infof("[gRPC] %T : %d -> %q (%v)", _ret, _ret.StatusCode, loc, took)
			} else {
				logger.Infof("[gRPC] %T : %d (%v)", _ret, _ret.StatusCode, took)
			}
		}
	case *filterapi.HTTPRequestModification:
		if _ret == nil {
			logger.Errorf("[gRPC] %T : unexpected nil (%v)", _ret, took)
		} else {
			logger.Infof("[gRPC] %T : %d headers (%v)", _ret, len(_ret.Header), took)
		}
	default:
		logger.Errorf("[gRPC] %T : unexpected response type (%v)", _ret, took)
	}
}

func (c *FilterMux) Filter(ctx context.Context, request *filterapi.FilterRequest) (ret filterapi.FilterResponse, err error) {
	start := time.Now()

	requestID := request.GetRequest().GetHttp().GetId()
	logger := c.Logger.WithField("REQUEST_ID", requestID)
	_ctx := middleware.WithRequestID(middleware.WithLogger(ctx, logger), requestID)

	logger.Infof("[gRPC] %s %s %s %s",
		request.GetRequest().GetHttp().GetProtocol(),
		request.GetRequest().GetHttp().GetMethod(),
		request.GetRequest().GetHttp().GetHost(),
		request.GetRequest().GetHttp().GetPath())
	defer func() {
		if _err := util.PanicToError(recover()); _err != nil {
			err = _err
		}
		if err != nil {
			ret = middleware.NewErrorResponse(_ctx, http.StatusInternalServerError, err, nil)
			err = nil
		}
		logResponse(logger, ret, time.Since(start))
	}()
	ret, err = c.filter(_ctx, request, requestID)
	return
}

func (c *FilterMux) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	switch r.URL.Path {
	case "/.ambassador/oauth2/logout":
		filterQName := r.FormValue("realm")
		filterInfo := findFilter(c.Controller, filterQName)
		if filterInfo == nil {
			middleware.ServeErrorResponse(w, ctx, http.StatusBadRequest,
				errors.Errorf("invalid realm: %q", filterQName), nil)
			return
		}
		filterSpec, filterSpecOK := filterInfo.Spec.(crd.FilterOAuth2)
		if !filterSpecOK {
			middleware.ServeErrorResponse(w, ctx, http.StatusBadRequest,
				errors.Errorf("invalid realm: %q", filterQName), nil)
			return
		}
		if filterInfo.Err != nil {
			middleware.ServeErrorResponse(w, ctx, http.StatusInternalServerError,
				errors.Wrapf(filterInfo.Err, "error in filter %q configuration", filterQName), nil)
			return
		}

		filterImpl := &oauth2handler.OAuth2Filter{
			PrivateKey: c.PrivateKey,
			PublicKey:  c.PublicKey,
			RedisPool:  c.RedisPool,
			QName:      filterQName,
			Spec:       filterSpec,
		}
		filterImpl.ServeHTTP(w, r)
	default:
		http.NotFound(w, r)
	}
}

func requestURL(request *filterapi.FilterRequest) (*url.URL, error) {
	var u *url.URL
	var err error

	str := request.GetRequest().GetHttp().GetPath()
	if request.GetRequest().GetHttp().GetMethod() == "CONNECT" && !strings.HasPrefix(str, "/") {
		u, err = url.ParseRequestURI("http://" + str)
		u.Scheme = ""
	} else {
		u, err = url.ParseRequestURI(str)
	}
	if err != nil {
		return nil, err
	}
	if u.Host == "" {
		u.Host = request.GetRequest().GetHttp().GetHost()
	}
	u.Scheme = request.GetRequest().GetHttp().GetScheme()

	return u, nil
}

func (c *FilterMux) filter(ctx context.Context, request *filterapi.FilterRequest, requestID string) (filterapi.FilterResponse, error) {
	logger := middleware.GetLogger(ctx)

	originalURL, err := requestURL(request)
	if err != nil {
		return nil, err
	}

	rule := ruleForURL(c.Controller, originalURL)
	if rule == nil {
		logger.Info("using default rule")
		rule = c.DefaultRule
	}
	filterStrs := make([]string, 0, len(rule.Filters))
	for _, filterRef := range rule.Filters {
		filterStrs = append(filterStrs, filterRef.Name+"."+filterRef.Namespace)
	}
	logger.Infof("selected rule host=%q, path=%q, filters=[%s]",
		rule.Host, rule.Path, strings.Join(filterStrs, ", "))

	return c.runFilters(rule.Filters, ctx, request)
}

func (c *FilterMux) runFilters(filters []crd.FilterReference, ctx context.Context, request *filterapi.FilterRequest) (filterapi.FilterResponse, error) {
	logger := middleware.GetLogger(ctx)

	sumResponse := &filterapi.HTTPRequestModification{}
	for _, filterRef := range filters {
		filterQName := filterRef.Name + "." + filterRef.Namespace
		if !filterRef.IfRequestHeader.Matches(filterutil.GetHeader(request)) {
			logger.Debugf("skipping filter=%q", filterQName)
			continue
		}
		logger.Debugf("applying filter=%q", filterQName)

		response, err := c.runFilter(filterRef, ctx, request)
		if err != nil {
			return nil, err
		}
		switch response := response.(type) {
		case *filterapi.HTTPResponse:
			switch filterRef.OnDeny {
			case crd.FilterActionBreak:
				return response, nil
			case crd.FilterActionContinue:
				// do nothing
			default:
				panic(errors.Errorf("unexpected filterRef.OnDeny: %q", filterRef.OnDeny))
			}
		case *filterapi.HTTPRequestModification:
			filterutil.ApplyRequestModification(request, response)
			sumResponse.Header = append(sumResponse.Header, response.Header...)
			switch filterRef.OnAllow {
			case crd.FilterActionBreak:
				return sumResponse, nil
			case crd.FilterActionContinue:
				// do nothing
			default:
				panic(errors.Errorf("unexpected filterRef.OnAllow: %q", filterRef.OnAllow))
			}
		default:
			panic(errors.Errorf("unexpected filter response type %T", response))
		}
	}
	return sumResponse, nil
}

func (c *FilterMux) runFilter(filterRef crd.FilterReference, ctx context.Context, request *filterapi.FilterRequest) (filterapi.FilterResponse, error) {
	filterQName := filterRef.Name + "." + filterRef.Namespace
	logger := middleware.GetLogger(ctx)

	filterInfo := findFilter(c.Controller, filterQName)
	if filterInfo == nil {
		return middleware.NewErrorResponse(ctx, http.StatusInternalServerError,
			errors.Errorf("could not find not filter: %q", filterQName), nil), nil
	}
	if filterInfo.Err != nil {
		return middleware.NewErrorResponse(ctx, http.StatusInternalServerError,
			errors.Wrapf(filterInfo.Err, "error in filter %q configuration", filterQName), nil), nil
	}

	var filterImpl filterapi.Filter
	switch filterSpec := filterInfo.Spec.(type) {
	case crd.FilterOAuth2:
		_filterImpl := &oauth2handler.OAuth2Filter{
			PrivateKey: c.PrivateKey,
			PublicKey:  c.PublicKey,
			RedisPool:  c.RedisPool,
			QName:      filterQName,
			Spec:       filterSpec,
			RunFilters: c.runFilters,
		}
		if err := mapstructure.Convert(filterRef.Arguments, &_filterImpl.Arguments); err != nil {
			return middleware.NewErrorResponse(ctx, http.StatusInternalServerError,
				errors.Wrap(err, "invalid filter.argument"), nil), nil
		}
		if err := _filterImpl.Arguments.Validate(filterRef.Namespace); err != nil {
			return middleware.NewErrorResponse(ctx, http.StatusInternalServerError,
				errors.Wrap(err, "invalid filter.argument"), nil), nil
		}
		filterImpl = _filterImpl
	case crd.FilterPlugin:
		filterImpl = filterutil.HandlerToFilter(filterSpec.Handler)
	case crd.FilterJWT:
		_filterImpl := &jwthandler.JWTFilter{
			Spec: filterSpec,
		}
		if err := mapstructure.Convert(filterRef.Arguments, &_filterImpl.Arguments); err != nil {
			return middleware.NewErrorResponse(ctx, http.StatusInternalServerError,
				errors.Wrap(err, "invalid filter.argument"), nil), nil
		}
		filterImpl = _filterImpl
	case crd.FilterExternal:
		filterImpl = &externalhandler.ExternalFilter{
			Spec: filterSpec,
		}
	case crd.FilterInternal:
		filterImpl = internalhandler.MakeInternalFilter()
	default:
		panic(errors.Errorf("unexpected filter type %T", filterSpec))
	}

	return filterImpl.Filter(middleware.WithLogger(ctx, logger.WithField("FILTER", filterQName)), request)
}

func ruleForURL(c *controller.Controller, u *url.URL) *crd.Rule {
	if u.Path == "/callback" {
		claims := jwt.MapClaims{}
		_, _, err := jwtsupport.SanitizeParseUnverified(new(jwt.Parser).ParseUnverified(u.Query().Get("state"), claims))
		if err == nil {
			if redirectURLstr, ok := claims["redirect_url"].(string); ok {
				_u, err := url.Parse(redirectURLstr)
				if err == nil {
					u = _u
				}
			}
		}
	}
	return findRule(c, u.Host, u.Path)
}

func findFilter(c *controller.Controller, qname string) *controller.FilterInfo {
	filters := c.LoadFilters()
	if filters == nil {
		return nil
	}
	filter, filterOK := filters[qname]
	if !filterOK {
		return nil
	}
	return &filter
}

func findRule(c *controller.Controller, host, path string) *crd.Rule {
	rules := c.LoadRules()
	if rules == nil {
		return nil
	}
	for _, rule := range rules {
		if rule.MatchHTTPHeaders(host, path) {
			return &rule
		}
	}
	return nil
}
