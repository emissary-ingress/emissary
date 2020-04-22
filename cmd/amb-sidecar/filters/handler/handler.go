package handler

import (
	"context"
	"crypto/rsa"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/datawire/ambassador/pkg/dlog"
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
	"github.com/datawire/apro/cmd/amb-sidecar/filters/handler/oauth2handler/client/authorization_code_client"
	"github.com/datawire/apro/cmd/amb-sidecar/limiter"
	"github.com/datawire/apro/lib/filterapi"
	"github.com/datawire/apro/lib/filterapi/filterutil"
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

// Filter implements the filterapi.Filter interface.  It is used to
// respond to (AuthService-configured) ext_authz gRPC calls.
func (c *FilterMux) Filter(ctx context.Context, request *filterapi.FilterRequest) (ret filterapi.FilterResponse, err error) {
	// This first part is boiler-plate of setting up last-defense
	// logging and panic recovery.

	start := time.Now()

	requestID := request.GetRequest().GetHttp().GetId()
	logger := c.Logger.WithField("REQUEST_ID", requestID)
	ctx = middleware.WithRequestID(dlog.WithLogger(ctx, logger), requestID)

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
			ret = middleware.NewErrorResponse(ctx, http.StatusInternalServerError, err, nil)
			err = nil
		}
		err = logResponse(logger, ret, time.Since(start))
		if err != nil {
			ret = middleware.NewErrorResponse(ctx, http.StatusInternalServerError, err, nil)
			err = nil
		}
	}()

	// The remainder is the meat of the function.

	originalURL, err := filterutil.GetURL(request)
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

// ServeHTTP implements the http.Handler interface.  It is used to
// respond to (Mapping-configured) HTTP calls.
//
// Having a Mapping pointing at the filter engine at all is kinda a
// kludge for Filters that require having their own bona fide HTTP
// endpoints.  We put up with that kludge because
//  - implementing a web server as a Filter should make your skin
//    crawl.
//  - whether or not our filters have access to the request body is an
//    awkwardly <https://github.com/datawire/apro/issues/903>
//    user-configurable thing, and we don't want to dictate the user's
//    settings.
//  - even after we implement #903, having our internal filters depend
//    on the request body would mean that Envoy would have to buffer
//    the request body for _all_ requests, which we want to avoid.
func (c *FilterMux) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	switch r.URL.Path {
	case "/.ambassador/oauth2/logout":
		// filterQName is the qualified name of the filter (name.namespace)
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

	case "/.ambassador/oauth2/multicookie":
		// Look up the `code` query parameter...
		mdState := r.URL.Query().Get("code")

		if mdState == "" {
			middleware.ServeErrorResponse(w, ctx, http.StatusBadRequest,
				errors.New("missing mdState"), nil)
			return
		}

		parts := strings.SplitN(mdState, ":", 2)
		if len(parts) != 2 {
			middleware.ServeErrorResponse(w, ctx, http.StatusBadRequest,
				errors.New("malformed mdState"), nil)
			return
		}

		filterQName := parts[0]

		filter := findFilter(c.Controller, filterQName)
		if filter == nil {
			middleware.ServeErrorResponse(w, ctx, http.StatusBadRequest,
				errors.Errorf("invalid mdState: %q", filterQName), nil)
			return
		}
		filterSpec, filterSpecOK := filter.UnwrappedSpec.(crd.FilterOAuth2)
		if !filterSpecOK {
			middleware.ServeErrorResponse(w, ctx, http.StatusBadRequest,
				errors.Errorf("invalid mdState: %q", filterQName), nil)
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
		// For historical reasons, the redirection-endpoint actually is implemented in Filter()
		// instead of ServeHTTP(); the only reason for it to stay that way is that it would be effort
		// to change it.
		//
		// So this code is just a fallback for when the redirect URL doesn't match any of the
		// configured Filters, so there's no sub-.Filter() to call.
		middleware.ServeErrorResponse(w, ctx, http.StatusBadRequest,
			errors.New("invalid state parameter; could not match to an OAuth2 Filter"), nil)
	default:
		http.NotFound(w, r)
	}
}

// runFilterRefs takes a list of FilterReferences, instantiates the
// appropriate Filter implementations, runs them on the given request,
// and aggregates the results; selecting which items to skip and how
// to aggregate based on the `ifRequestHeader`, `onDeny`, and
// `onAllow` fields in each FilterReference.
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

// runFilterRef takes a reference to a Filter (which for the purposes
// of this method you can think of as a "(name, namespace, arguments)"
// tuple), instantiates the appropriate Filter implementation, and
// runs it on the given request.
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
		filterImpl, errResponse = c.getFilterImpl(ctx, filter, filterRef.Arguments)
		if errResponse != nil {
			return errResponse, nil
		}
	}

	return filterImpl.Filter(dlog.WithLogger(ctx, dlog.GetLogger(ctx).WithField("FILTER", filterQName)), request)
}

// runJWTFilterRef is *almost* a copy of c.runFilterRef, but
//  1. clarifies that this is a JWT-sub-filter in log/error messages, and
//  2. validates that it's a JWT filter, and not a filter of another type.
func (c *FilterMux) runJWTFilterRef(filterRef crd.JWTFilterReference, ctx context.Context, request *filterapi.FilterRequest) (filterapi.FilterResponse, error) {
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
	filterImpl, errResponse := c.getFilterImpl(ctx, filter, filterRef.Arguments)
	if errResponse != nil {
		return errResponse, nil
	}

	return filterImpl.Filter(dlog.WithLogger(ctx, dlog.GetLogger(ctx).WithField("JWTFILTER", filterQName)), request)
}

// getFilterImpl takes a Filter CRD object and any path-specific
// arguments configured in the FilterPolicy, and returns an
// instantiated implementation of that Filter: (filterImpl, nil).
//
// If there is an error instantiating the Filter (perhaps there is an
// error in the Filter definition, or in the arguments), then an error
// response object is returned: (nil, errorResponse).
func (c *FilterMux) getFilterImpl(ctx context.Context, filter *crd.Filter, filterArguments interface{}) (filterapi.Filter, filterapi.FilterResponse) {
	filterQName := filter.GetName() + "." + filter.GetNamespace()
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
		if err := _filterImpl.Arguments.Validate(filter.GetNamespace()); err != nil {
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

// syntheticRule creates an internal-only Rule, rather than one that
// is from a FilterPolicy configuration.  A normal configured Rule
// refers to a Filter by name/namespace; one of these synthetic rules
// allows you to pass in the instantiated Filter implementation
// directly, and avoid having to come up with some bogus magic name
// for it.
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

// ruleForURL returns the appropriate Rule (from a FilterPolicy
// resource) for a given URL.  That Rule says which Filters to call
// for this request.
//
// Besides looking up Rules based on configured FilterPolicies, it
// also handles a few hard-coded special-cases.
//
// A nil return value means that there is no Rule associated with this
// URL; and to use c.DefaultRule (currently set in app.go; "don't
// apply any Filters to this request").
func (c *FilterMux) ruleForURL(u *url.URL) *crd.Rule {
	// First-up: check for the special-cases
	switch u.Path {
	case "/.ambassador/oauth2/logout":
		return nil
	case "/.ambassador/oauth2/redirection-endpoint":
		// For historical reasons, the redirection-endpoint actually is implemented in Filter()
		// instead of ServeHTTP(); the only reason for it to stay that way is that it would be effort
		// to change it.
		//
		// If that ever does change, this needs to be changed to just return nil.
		_u, err := authorization_code_client.ReadState(u.Query().Get("state"))
		if err == nil {
			u = _u
		}
		// Continue to the FilterPolicy-based common-case below; after the end of the switch-block
		// (i.e. yes, it's intentional that there's no `return` here).
	default:
		switch {
		case strings.HasPrefix(u.Path, "/.well-known/acme-challenge/"):
			if c.RedisPool == nil {
				return nil
			}
			return syntheticRule(acmeclient.NewChallengeHandler(c.RedisPool))
		case strings.Contains(u.Path, "/.ambassador-internal/"):
			return syntheticRule(devportalfilter.MakeDevPortalFilter())
		}
	}

	// OK, this is the FilterPolicy-based common-case where we
	// look up the Rule based on the configured FilterPolicies.
	_, rules := c.Controller.LoadPolicies()
	if rules == nil {
		return nil
	}
	for _, rule := range rules {
		if rule.MatchHTTPHeaders(u.Host, u.Path) {
			return &rule
		}
	}
	return nil
}

// findFilter returns the Filter CRD object with a the given qualified
// name ("name.namespace").  It returns nil if no Filter with that
// qualified name exists.
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
