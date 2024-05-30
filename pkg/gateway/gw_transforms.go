package gateway

import (
	// standard library
	"fmt"

	// third-party libraries
	"github.com/pkg/errors"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/wrapperspb"
	gw "sigs.k8s.io/gateway-api/apis/v1alpha1"

	// envoy api v3
	v3core "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	v3listener "github.com/envoyproxy/go-control-plane/envoy/config/listener/v3"
	v3route "github.com/envoyproxy/go-control-plane/envoy/config/route/v3"
	v3httpman "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/network/http_connection_manager/v3"
	v3matcher "github.com/envoyproxy/go-control-plane/envoy/type/matcher/v3"

	// envoy control plane
	ecp_wellknown "github.com/envoyproxy/go-control-plane/pkg/wellknown"

	// first-party libraries
	"github.com/emissary-ingress/emissary/v3/pkg/kates"
)

func Compile_Gateway(gateway *gw.Gateway) (*CompiledConfig, error) {
	src := SourceFromResource(gateway)
	var listeners []*CompiledListener
	for idx, l := range gateway.Spec.Listeners {
		name := fmt.Sprintf("%s-%d", getName(gateway), idx)
		listener, err := Compile_Listener(src, l, name)
		if err != nil {
			return nil, err
		}
		listeners = append(listeners, listener)
	}
	return &CompiledConfig{
		CompiledItem: NewCompiledItem(src),
		Listeners:    listeners,
	}, nil
}

func Compile_Listener(parent Source, lst gw.Listener, name string) (*CompiledListener, error) {
	hcm := &v3httpman.HttpConnectionManager{
		StatPrefix: name,
		HttpFilters: []*v3httpman.HttpFilter{
			{Name: ecp_wellknown.CORS},
			{Name: ecp_wellknown.Router},
		},
		RouteSpecifier: &v3httpman.HttpConnectionManager_Rds{
			Rds: &v3httpman.Rds{
				ConfigSource: &v3core.ConfigSource{
					ConfigSourceSpecifier: &v3core.ConfigSource_Ads{
						Ads: &v3core.AggregatedConfigSource{},
					},
				},
				RouteConfigName: name,
			},
		},
	}
	hcmAny, err := anypb.New(hcm)
	if err != nil {
		return nil, err
	}

	return &CompiledListener{
		CompiledItem: NewCompiledItem(Sourcef("listener %s", lst.Hostname)),
		Listener: &v3listener.Listener{
			Name: name,
			Address: &v3core.Address{Address: &v3core.Address_SocketAddress{SocketAddress: &v3core.SocketAddress{
				Address:       "0.0.0.0",
				PortSpecifier: &v3core.SocketAddress_PortValue{PortValue: uint32(lst.Port)},
			}}},
			FilterChains: []*v3listener.FilterChain{
				{
					Filters: []*v3listener.Filter{
						{
							Name:       ecp_wellknown.HTTPConnectionManager,
							ConfigType: &v3listener.Filter_TypedConfig{TypedConfig: hcmAny},
						},
					},
				},
			},
		},
		Predicate: func(route *CompiledRoute) bool {
			return true
		},
		Domains: []string{"*"},
	}, nil

}

func Compile_HTTPRoute(httpRoute *gw.HTTPRoute) (*CompiledConfig, error) {
	src := SourceFromResource(httpRoute)
	clusterRefs := []*ClusterRef{}
	var routes []*v3route.Route
	for idx, rule := range httpRoute.Spec.Rules {
		s := Sourcef("rule %d in %s", idx, src)
		_routes, err := Compile_HTTPRouteRule(s, rule, httpRoute.Namespace, &clusterRefs)
		if err != nil {
			return nil, err
		}
		routes = append(routes, _routes...)
	}
	return &CompiledConfig{
		CompiledItem: NewCompiledItem(src),
		Routes: []*CompiledRoute{
			{
				CompiledItem: CompiledItem{Source: src, Namespace: httpRoute.Namespace},
				HTTPRoute:    httpRoute,
				Routes:       routes,
				ClusterRefs:  clusterRefs,
			},
		},
	}, nil
}

func Compile_HTTPRouteRule(src Source, rule gw.HTTPRouteRule, namespace string, clusterRefs *[]*ClusterRef) ([]*v3route.Route, error) {
	var clusters []*v3route.WeightedCluster_ClusterWeight
	for idx, fwd := range rule.ForwardTo {
		s := Sourcef("forwardTo %d in %s", idx, src)
		clusters = append(clusters, Compile_HTTPRouteForwardTo(s, fwd, namespace, clusterRefs))
	}

	wc := &v3route.WeightedCluster{Clusters: clusters}

	matches, err := Compile_HTTPRouteMatches(rule.Matches)
	if err != nil {
		return nil, err
	}
	var result []*v3route.Route
	for _, match := range matches {
		result = append(result, &v3route.Route{
			Match: match,
			Action: &v3route.Route_Route{Route: &v3route.RouteAction{
				ClusterSpecifier: &v3route.RouteAction_WeightedClusters{WeightedClusters: wc},
			}},
		})
	}

	return result, err
}

func Compile_HTTPRouteForwardTo(src Source, forward gw.HTTPRouteForwardTo, namespace string, clusterRefs *[]*ClusterRef) *v3route.WeightedCluster_ClusterWeight {
	suffix := ""
	clusterName := *forward.ServiceName
	if forward.Port != nil {
		suffix = fmt.Sprintf("/%d", *forward.Port)
		clusterName = fmt.Sprintf("%s_%d", *forward.ServiceName, *forward.Port)
	}

	*clusterRefs = append(*clusterRefs, &ClusterRef{
		CompiledItem: NewCompiledItem(src),
		Name:         clusterName,
		EndpointPath: fmt.Sprintf("k8s/%s/%s%s", namespace, *forward.ServiceName, suffix),
	})
	return &v3route.WeightedCluster_ClusterWeight{
		Name:   clusterName,
		Weight: &wrapperspb.UInt32Value{Value: uint32(forward.Weight)},
	}
}

func Compile_HTTPRouteMatches(matches []gw.HTTPRouteMatch) ([]*v3route.RouteMatch, error) {
	var result []*v3route.RouteMatch
	for _, match := range matches {
		item, err := Compile_HTTPRouteMatch(match)
		if err != nil {
			return nil, err
		}
		result = append(result, item)
	}
	return result, nil
}

func Compile_HTTPRouteMatch(match gw.HTTPRouteMatch) (*v3route.RouteMatch, error) {
	headers, err := Compile_HTTPHeaderMatch(match.Headers)
	if err != nil {
		return nil, err
	}
	result := &v3route.RouteMatch{
		Headers: headers,
	}

	switch match.Path.Type {
	case gw.PathMatchExact:
		result.PathSpecifier = &v3route.RouteMatch_Path{Path: match.Path.Value}
	case gw.PathMatchPrefix:
		result.PathSpecifier = &v3route.RouteMatch_Prefix{Prefix: match.Path.Value}
	case gw.PathMatchRegularExpression:
		result.PathSpecifier = &v3route.RouteMatch_SafeRegex{SafeRegex: regexMatcher(match.Path.Value)}
	case "":
		// no path match, but PathSpecifier is required
		result.PathSpecifier = &v3route.RouteMatch_Prefix{}
	default:
		return nil, errors.Errorf("unknown path match type: %q", match.Path.Type)
	}

	return result, nil
}

func Compile_HTTPHeaderMatch(headerMatch *gw.HTTPHeaderMatch) ([]*v3route.HeaderMatcher, error) {
	if headerMatch == nil {
		return nil, nil
	}

	var result []*v3route.HeaderMatcher
	for hdr, pattern := range headerMatch.Values {
		hm := &v3route.HeaderMatcher{
			Name:        hdr,
			InvertMatch: false,
		}

		switch headerMatch.Type {
		case gw.HeaderMatchExact:
			hm.HeaderMatchSpecifier = &v3route.HeaderMatcher_ExactMatch{ExactMatch: pattern}
		case gw.HeaderMatchRegularExpression:
			hm.HeaderMatchSpecifier = &v3route.HeaderMatcher_SafeRegexMatch{SafeRegexMatch: regexMatcher(pattern)}
		default:
			return nil, errors.Errorf("unknown header match type: %s", headerMatch.Type)
		}

		result = append(result, hm)
	}
	return result, nil
}

func regexMatcher(pattern string) *v3matcher.RegexMatcher {
	return &v3matcher.RegexMatcher{
		EngineType: &v3matcher.RegexMatcher_GoogleRe2{GoogleRe2: &v3matcher.RegexMatcher_GoogleRE2{}},
		Regex:      pattern,
	}
}

func getName(resource kates.Object) string {
	return fmt.Sprintf("%s-%s", resource.GetNamespace(), resource.GetName())
}
