package app

import (
	"context"
	"crypto/rsa"
	"encoding/json"
	"net/http"
	"net/url"
	"runtime"
	"strings"
	"time"

	jwt "github.com/dgrijalva/jwt-go"
	"github.com/pkg/errors"

	crd "github.com/datawire/apro/apis/getambassador.io/v1beta2"
	"github.com/datawire/apro/cmd/amb-sidecar/filters/app/externalhandler"
	"github.com/datawire/apro/cmd/amb-sidecar/filters/app/jwthandler"
	"github.com/datawire/apro/cmd/amb-sidecar/filters/app/middleware"
	"github.com/datawire/apro/cmd/amb-sidecar/filters/app/oauth2handler"
	"github.com/datawire/apro/cmd/amb-sidecar/filters/controller"
	"github.com/datawire/apro/cmd/amb-sidecar/types"
	"github.com/datawire/apro/lib/filterapi"
	"github.com/datawire/apro/lib/filterapi/filterutil"
	"github.com/datawire/apro/lib/mapstructure"
)

type FilterMux struct {
	Controller  *controller.Controller
	DefaultRule *crd.Rule
	PrivateKey  *rsa.PrivateKey
	PublicKey   *rsa.PublicKey
	Logger      types.Logger
}

func errorResponse(httpStatus int, err error, requestID string, logger types.Logger) *filterapi.HTTPResponse {
	body := map[string]interface{}{
		"status_code": httpStatus,
		"message":     err.Error(),
	}
	if httpStatus/100 == 5 {
		body["request_id"] = requestID
	}
	bodyBytes, _ := json.Marshal(body)
	logger.Infoln(httpStatus, err)
	return &filterapi.HTTPResponse{
		StatusCode: httpStatus,
		Header: http.Header{
			"Content-Type": {"application/json"},
		},
		Body: string(bodyBytes),
	}
}

func logResponse(logger types.Logger, ret filterapi.FilterResponse, took time.Duration) {
	switch _ret := ret.(type) {
	case *filterapi.HTTPResponse:
		if loc := _ret.Header.Get("Location"); loc != "" {
			logger.Infof("[gRPC] %T : %d -> %q (%v)", _ret, _ret.StatusCode, loc, took)
		} else {
			logger.Infof("[gRPC] %T : %d (%v)", _ret, _ret.StatusCode, took)
		}
	case *filterapi.HTTPRequestModification:
		logger.Infof("[gRPC] %T : %d headers (%v)", _ret, len(_ret.Header), took)
	default:
		logger.Infof("[gRPC] %T : unexpected response type (%v)", _ret, took)
	}
}

func (c *FilterMux) Filter(ctx context.Context, request *filterapi.FilterRequest) (ret filterapi.FilterResponse, err error) {
	start := time.Now()
	requestID := request.GetRequest().GetHttp().GetId()
	logger := c.Logger.WithField("REQUEST_ID", requestID)
	logger.Infof("[gRPC] %s %s %s %s",
		request.GetRequest().GetHttp().GetProtocol(),
		request.GetRequest().GetHttp().GetMethod(),
		request.GetRequest().GetHttp().GetHost(),
		request.GetRequest().GetHttp().GetPath())
	defer func() {
		if rec := recover(); rec != nil {
			const stacksize = 64 << 10 // net/http uses 64<<10, negroni.Recovery uses 1024*8 by default
			stack := make([]byte, stacksize)
			stack = stack[:runtime.Stack(stack, false)]
			logger.Errorf("PANIC: %v\n%s", rec, stack)

			err = errors.Errorf("PANIC: %v", rec)
		}
		if err != nil {
			ret = errorResponse(http.StatusInternalServerError, err, requestID, logger)
			err = nil
		}
		logResponse(logger, ret, time.Since(start))
	}()
	ret, err = c.filter(middleware.WithLogger(ctx, logger), request, requestID)
	return
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

	sumResponse := &filterapi.HTTPRequestModification{}
	for _, filterRef := range rule.Filters {
		filterQName := filterRef.Name + "." + filterRef.Namespace
		logger.Debugf("applying filter=%q", filterQName)

		filterCRD := findFilter(c.Controller, filterQName)
		if filterCRD == nil {
			return errorResponse(http.StatusInternalServerError, errors.Errorf("could not find not filter: %q", filterQName), requestID, logger), nil
		}

		var filterImpl filterapi.Filter
		switch filterCRD := filterCRD.(type) {
		case crd.FilterOAuth2:
			handler := &oauth2handler.OAuth2Handler{
				PrivateKey: c.PrivateKey,
				PublicKey:  c.PublicKey,
				Filter:     filterCRD,
			}
			if err := mapstructure.Convert(filterRef.Arguments, &handler.FilterArguments); err != nil {
				return errorResponse(http.StatusInternalServerError, errors.Wrap(err, "invalid filter.argument"), requestID, logger), nil
			}
			filterImpl = filterutil.HandlerToFilter(handler)
		case crd.FilterPlugin:
			filterImpl = filterutil.HandlerToFilter(filterCRD.Handler)
		case crd.FilterJWT:
			filterImpl = filterutil.HandlerToFilter(&jwthandler.JWTHandler{
				Filter: filterCRD,
			})
		case crd.FilterExternal:
			filterImpl = &externalhandler.ExternalFilter{
				Spec: filterCRD,
			}
		default:
			panic(errors.Errorf("unexpected filter type %T", filterCRD))
		}

		response, err := filterImpl.Filter(middleware.WithLogger(ctx, logger.WithField("FILTER", filterQName)), request)
		if err != nil {
			return nil, err
		}
		switch response := response.(type) {
		case *filterapi.HTTPResponse:
			return response, nil
		case *filterapi.HTTPRequestModification:
			handleRequestModification(request, response)
			sumResponse.Header = append(sumResponse.Header, response.Header...)
		default:
			panic(errors.Errorf("unexpexted filter response type %T", response))
		}
	}
	return sumResponse, nil
}

func handleRequestModification(req *filterapi.FilterRequest, mod *filterapi.HTTPRequestModification) {
	for _, hmod := range mod.Header {
		switch hmod := hmod.(type) {
		case *filterapi.HTTPHeaderAppendValue:
			if cur, ok := req.Request.Http.Headers[http.CanonicalHeaderKey(hmod.Key)]; ok {
				req.Request.Http.Headers[http.CanonicalHeaderKey(hmod.Key)] = cur + "," + hmod.Value
			} else {
				req.Request.Http.Headers[http.CanonicalHeaderKey(hmod.Key)] = hmod.Value
			}
		case *filterapi.HTTPHeaderReplaceValue:
			req.Request.Http.Headers[http.CanonicalHeaderKey(hmod.Key)] = hmod.Value
		default:
			panic(errors.Errorf("unexpected header modification type %T", hmod))
		}
	}
}

func ruleForURL(c *controller.Controller, u *url.URL) *crd.Rule {
	if u.Path == "/callback" {
		claims := jwt.MapClaims{}
		_, _, err := new(jwt.Parser).ParseUnverified(u.Query().Get("state"), claims)
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
