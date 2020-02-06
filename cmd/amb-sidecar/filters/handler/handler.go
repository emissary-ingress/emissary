package handler

import (
	"context"
	"crypto/rsa"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/datawire/ambassador/pkg/dlog"
	"github.com/dgrijalva/jwt-go"
	"github.com/mediocregopher/radix.v2/pool"
	"github.com/pkg/errors"

	crd "github.com/datawire/apro/apis/getambassador.io/v1beta2"
	"github.com/datawire/apro/cmd/amb-sidecar/acmeclient"
	"github.com/datawire/apro/cmd/amb-sidecar/devportal/devportalfilter"
	"github.com/datawire/apro/cmd/amb-sidecar/filters/controller"
	"github.com/datawire/apro/cmd/amb-sidecar/filters/handler/externalhandler"
	"github.com/datawire/apro/cmd/amb-sidecar/filters/handler/jwthandler"
	"github.com/datawire/apro/cmd/amb-sidecar/filters/handler/middleware"
	"github.com/datawire/apro/cmd/amb-sidecar/filters/handler/oauth2handler"
	"github.com/datawire/apro/cmd/amb-sidecar/limiter"
	"github.com/datawire/apro/lib/filterapi"
	"github.com/datawire/apro/lib/filterapi/filterutil"
	"github.com/datawire/apro/lib/jwtsupport"
	"github.com/datawire/apro/lib/mapstructure"
	"github.com/datawire/apro/lib/util"
)

type FilterMux struct {
	Controller      *controller.Controller
	DefaultRule     *crd.Rule
	PrivateKey      *rsa.PrivateKey
	PublicKey       *rsa.PublicKey
	Logger          dlog.Logger
	RedisPool       *pool.Pool
	AuthRateLimiter limiter.RateLimiter
}

func logResponse(logger dlog.Logger, ret filterapi.FilterResponse, took time.Duration) error {
	switch _ret := ret.(type) {
	case nil:
		err := errors.Errorf("[gRPC] %T : unexpected nil", _ret)
		logger.Errorf("%v (%v)", err, took)
		return err
	case *filterapi.HTTPResponse:
		if _ret == nil {
			err := errors.Errorf("[gRPC] %T : unexpected nil", _ret)
			logger.Errorf("%v (%v)", err, took)
			return err
		} else {
			if loc := _ret.Header.Get("Location"); loc != "" {
				logger.Infof("[gRPC] %T : %d -> %q (%v)", _ret, _ret.StatusCode, loc, took)
			} else {
				logger.Infof("[gRPC] %T : %d (%v)", _ret, _ret.StatusCode, took)
			}
		}
	case *filterapi.HTTPRequestModification:
		if _ret == nil {
			err := errors.Errorf("[gRPC] %T : unexpected nil", _ret)
			logger.Errorf("%v (%v)", err, took)
			return err
		} else {
			logger.Infof("[gRPC] %T : %d headers (%v)", _ret, len(_ret.Header), took)
		}
	default:
		err := errors.Errorf("[gRPC] %T : unexpected response type", _ret)
		logger.Errorf("%v (%v)", err, took)
		return err
	}
	return nil
}

func (c *FilterMux) Filter(ctx context.Context, request *filterapi.FilterRequest) (ret filterapi.FilterResponse, err error) {
	start := time.Now()

	requestID := request.GetRequest().GetHttp().GetId()
	logger := c.Logger.WithField("REQUEST_ID", requestID)
	_ctx := middleware.WithRequestID(dlog.WithLogger(ctx, logger), requestID)

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
		err = logResponse(logger, ret, time.Since(start))
		if err != nil {
			ret = middleware.NewErrorResponse(_ctx, http.StatusInternalServerError, err, nil)
			err = nil
		}
	}()
	ret, err = c.filter(_ctx, request, requestID)
	return
}

func (c *FilterMux) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	switch r.URL.Path {
	case "/.ambassador/oauth2/logout":
		filterQName := r.FormValue("realm")
		filter := findFilter(c.Controller, filterQName)
		if filter == nil {
			middleware.ServeErrorResponse(w, ctx, http.StatusBadRequest,
				errors.Errorf("invalid realm: %q", filterQName), nil)
			return
		}
		filterSpec, filterSpecOK := filter.UnwrappedSpec.(crd.FilterOAuth2)
		if !filterSpecOK {
			middleware.ServeErrorResponse(w, ctx, http.StatusBadRequest,
				errors.Errorf("invalid realm: %q", filterQName), nil)
			return
		}
		if filter.Status.State != crd.FilterState_OK {
			middleware.ServeErrorResponse(w, ctx, http.StatusInternalServerError,
				errors.Errorf("error in filter %q configuration: %s", filterQName, filter.Status.Reason), nil)
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
	case "/.ambassador/oauth2/redirection-endpoint":
		middleware.ServeErrorResponse(w, ctx, http.StatusBadRequest,
			errors.New("invalid state parameter; could not match to an OAuth2 Filter"), nil)
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
	logger := dlog.GetLogger(ctx)

	originalURL, err := requestURL(request)
	if err != nil {
		return nil, err
	}

	rule := c.ruleForURL(originalURL)
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

	return c.runFilterRefs(rule.Filters, ctx, request)
}

func (c *FilterMux) runFilterRefs(filters []crd.FilterReference, ctx context.Context, request *filterapi.FilterRequest) (filterapi.FilterResponse, error) {
	logger := dlog.GetLogger(ctx)

	sumResponse := &filterapi.HTTPRequestModification{}
	for _, filterRef := range filters {
		filterQName := filterRef.Name + "." + filterRef.Namespace
		if !filterRef.IfRequestHeader.Matches(filterutil.GetHeader(request)) {
			logger.Debugf("skipping filter=%q", filterQName)
			continue
		}
		logger.Debugf("applying filter=%q", filterQName)

		response, err := c.runFilterRef(filterRef, ctx, request)
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

func (c *FilterMux) runFilterRef(filterRef crd.FilterReference, ctx context.Context, request *filterapi.FilterRequest) (filterapi.FilterResponse, error) {
	filterQName := filterRef.Name + "." + filterRef.Namespace

	var filterImpl filterapi.Filter
	if filterRef.Impl != nil {
		filterImpl = filterRef.Impl
	} else {
		filter := findFilter(c.Controller, filterQName)
		if filter == nil {
			return middleware.NewErrorResponse(ctx, http.StatusInternalServerError,
				errors.Errorf("could not find not filter: %q", filterQName), nil), nil
		}
		if filter.Status.State != crd.FilterState_OK {
			return middleware.NewErrorResponse(ctx, http.StatusInternalServerError,
				errors.Errorf("error in filter %q configuration: %s", filterQName, filter.Status.Reason), nil), nil
		}
		var errResponse filterapi.FilterResponse
		filterImpl, errResponse = c.getFilterImpl(filter, filterRef.Name, filterRef.Namespace, filterRef.Arguments, ctx)
		if errResponse != nil {
			return errResponse, nil
		}
	}

	return filterImpl.Filter(dlog.WithLogger(ctx, dlog.GetLogger(ctx).WithField("FILTER", filterQName)), request)
}

func (c *FilterMux) runJWTFilterRef(filterRef crd.JWTFilterReference, ctx context.Context, request *filterapi.FilterRequest) (filterapi.FilterResponse, error) {
	// This is *almost* a copy of c.runFilterRef, but
	//  1. clarifies that this is a JWT-sub-filter in log/error messages, and
	//  2. validates that it's a JWT filter, and not a filter of another type.
	filterQName := filterRef.Name + "." + filterRef.Namespace

	filter := findFilter(c.Controller, filterQName)
	if filter == nil {
		return middleware.NewErrorResponse(ctx, http.StatusInternalServerError,
			errors.Errorf("could not find not JWT filter: %q", filterQName), nil), nil
	}
	if _, isJWT := filter.UnwrappedSpec.(crd.FilterJWT); !isJWT {
		return middleware.NewErrorResponse(ctx, http.StatusInternalServerError,
			errors.Errorf("filter %q is not a JWT filter", filterQName), nil), nil
	}
	if filter.Status.State != crd.FilterState_OK {
		return middleware.NewErrorResponse(ctx, http.StatusInternalServerError,
			errors.Errorf("error in JWT filter %q configuration: %s", filterQName, filter.Status.Reason), nil), nil
	}
	filterImpl, errResponse := c.getFilterImpl(filter, filterRef.Name, filterRef.Namespace, filterRef.Arguments, ctx)
	if errResponse != nil {
		return errResponse, nil
	}

	return filterImpl.Filter(dlog.WithLogger(ctx, dlog.GetLogger(ctx).WithField("JWTFILTER", filterQName)), request)
}

func (c *FilterMux) getFilterImpl(filter *crd.Filter, filterName, filterNamespace string, filterArguments interface{}, ctx context.Context) (filterapi.Filter, filterapi.FilterResponse) {
	filterQName := filterName + "." + filterNamespace
	var filterImpl filterapi.Filter
	switch filterSpec := filter.UnwrappedSpec.(type) {
	case crd.FilterOAuth2:
		err := c.AuthRateLimiter.IncrementUsage()
		if err != nil {
			if err == limiter.ErrRateLimiterNoRedis {
				return nil, middleware.NewErrorResponse(ctx, http.StatusInternalServerError, err, nil)
			}
			return nil, middleware.NewErrorResponse(ctx, http.StatusTooManyRequests, err, nil)
		}

		_filterImpl := &oauth2handler.OAuth2Filter{
			PrivateKey:   c.PrivateKey,
			PublicKey:    c.PublicKey,
			RedisPool:    c.RedisPool,
			QName:        filterQName,
			Spec:         filterSpec,
			RunFilters:   c.runFilterRefs,
			RunJWTFilter: c.runJWTFilterRef,
		}
		if err := mapstructure.Convert(filterArguments, &_filterImpl.Arguments); err != nil {
			return nil, middleware.NewErrorResponse(ctx, http.StatusInternalServerError,
				errors.Wrap(err, "invalid filter.argument"), nil)
		}
		if err := _filterImpl.Arguments.Validate(filterNamespace); err != nil {
			return nil, middleware.NewErrorResponse(ctx, http.StatusInternalServerError,
				errors.Wrap(err, "invalid filter.argument"), nil)
		}
		filterImpl = _filterImpl
	case crd.FilterPlugin:
		filterImpl = filterutil.HandlerToFilter(filterSpec.Handler)
	case crd.FilterJWT:
		err := c.AuthRateLimiter.IncrementUsage()
		if err != nil {
			if err == limiter.ErrRateLimiterNoRedis {
				return nil, middleware.NewErrorResponse(ctx, http.StatusInternalServerError, err, nil)
			}
			return nil, middleware.NewErrorResponse(ctx, http.StatusTooManyRequests, err, nil)
		}

		_filterImpl := &jwthandler.JWTFilter{
			Spec: filterSpec,
		}
		if err := mapstructure.Convert(filterArguments, &_filterImpl.Arguments); err != nil {
			return nil, middleware.NewErrorResponse(ctx, http.StatusInternalServerError,
				errors.Wrap(err, "invalid filter.argument"), nil)
		}
		filterImpl = _filterImpl
	case crd.FilterExternal:
		filterImpl = &externalhandler.ExternalFilter{
			Spec: filterSpec,
		}
	default:
		panic(errors.Errorf("unexpected filter type %T", filterSpec))
	}

	return filterImpl, nil
}

func syntheticRule(filterImpl filterapi.Filter) *crd.Rule {
	ret := &crd.Rule{
		Filters: []crd.FilterReference{
			{Impl: filterImpl},
		},
	}
	if err := ret.Validate(""); err != nil {
		// This should never happen; the Rule we created above
		// should be valid.
		panic(err)
	}
	return ret
}

func (c *FilterMux) ruleForURL(u *url.URL) *crd.Rule {
	switch u.Path {
	case "/.ambassador/oauth2/logout":
		return nil
	case "/.ambassador/oauth2/redirection-endpoint":
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
		return findRule(c.Controller, u.Host, u.Path)
	default:
		switch {
		case strings.HasPrefix(u.Path, "/.well-known/acme-challenge/"):
			if c.RedisPool == nil {
				return nil
			}
			return syntheticRule(acmeclient.NewChallengeHandler(c.RedisPool))
		case strings.Contains(u.Path, "/.ambassador-internal/"):
			return syntheticRule(devportalfilter.MakeDevPortalFilter())
		default:
			return findRule(c.Controller, u.Host, u.Path)
		}
	}
}

func findFilter(c *controller.Controller, qname string) *crd.Filter {
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
	_, rules := c.LoadPolicies()
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
