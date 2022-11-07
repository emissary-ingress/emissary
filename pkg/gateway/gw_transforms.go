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
	apiv3_core "github.com/emissary-ingress/emissary/v3/pkg/api/envoy/config/core/v3"
	apiv3_listener "github.com/emissary-ingress/emissary/v3/pkg/api/envoy/config/listener/v3"
	apiv3_route "github.com/emissary-ingress/emissary/v3/pkg/api/envoy/config/route/v3"
	apiv3_httpman "github.com/emissary-ingress/emissary/v3/pkg/api/envoy/extensions/filters/network/http_connection_manager/v3"
	apiv3_matcher "github.com/emissary-ingress/emissary/v3/pkg/api/envoy/type/matcher/v3"

	// envoy control plane
	ecp_wellknown "github.com/emissary-ingress/emissary/v3/pkg/envoy-control-plane/wellknown"

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
	hcm := &apiv3_httpman.HttpConnectionManager{
		StatPrefix: name,
		HttpFilters: []*apiv3_httpman.HttpFilter{
			{Name: ecp_wellknown.CORS},
			{Name: ecp_wellknown.Router},
		},
		RouteSpecifier: &apiv3_httpman.HttpConnectionManager_Rds{
			Rds: &apiv3_httpman.Rds{
				ConfigSource: &apiv3_core.ConfigSource{
					ConfigSourceSpecifier: &apiv3_core.ConfigSource_Ads{
						Ads: &apiv3_core.AggregatedConfigSource{},
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
		Listener: &apiv3_listener.Listener{
			Name: name,
			Address: &apiv3_core.Address{Address: &apiv3_core.Address_SocketAddress{SocketAddress: &apiv3_core.SocketAddress{
				Address:       "0.0.0.0",
				PortSpecifier: &apiv3_core.SocketAddress_PortValue{PortValue: uint32(lst.Port)},
			}}},
			FilterChains: []*apiv3_listener.FilterChain{
				{
					Filters: []*apiv3_listener.Filter{
						{
							Name:       ecp_wellknown.HTTPConnectionManager,
							ConfigType: &apiv3_listener.Filter_TypedConfig{TypedConfig: hcmAny},
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
	var routes []*apiv3_route.Route
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

func Compile_HTTPRouteRule(src Source, rule gw.HTTPRouteRule, namespace string, clusterRefs *[]*ClusterRef) ([]*apiv3_route.Route, error) {
	var clusters []*apiv3_route.WeightedCluster_ClusterWeight
	for idx, fwd := range rule.ForwardTo {
		s := Sourcef("forwardTo %d in %s", idx, src)
		clusters = append(clusters, Compile_HTTPRouteForwardTo(s, fwd, namespace, clusterRefs))
	}

	wc := &apiv3_route.WeightedCluster{Clusters: clusters}

	matches, err := Compile_HTTPRouteMatches(rule.Matches)
	if err != nil {
		return nil, err
	}
	var result []*apiv3_route.Route
	for _, match := range matches {
		result = append(result, &apiv3_route.Route{
			Match: match,
			Action: &apiv3_route.Route_Route{Route: &apiv3_route.RouteAction{
				ClusterSpecifier: &apiv3_route.RouteAction_WeightedClusters{WeightedClusters: wc},
			}},
		})
	}

	return result, err
}

func Compile_HTTPRouteForwardTo(src Source, forward gw.HTTPRouteForwardTo, namespace string, clusterRefs *[]*ClusterRef) *apiv3_route.WeightedCluster_ClusterWeight {
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
	return &apiv3_route.WeightedCluster_ClusterWeight{
		Name:   clusterName,
		Weight: &wrapperspb.UInt32Value{Value: uint32(forward.Weight)},
	}
}

func Compile_HTTPRouteMatches(matches []gw.HTTPRouteMatch) ([]*apiv3_route.RouteMatch, error) {
	var result []*apiv3_route.RouteMatch
	for _, match := range matches {
		item, err := Compile_HTTPRouteMatch(match)
		if err != nil {
			return nil, err
		}
		result = append(result, item)
	}
	return result, nil
}

func Compile_HTTPRouteMatch(match gw.HTTPRouteMatch) (*apiv3_route.RouteMatch, error) {
	headers, err := Compile_HTTPHeaderMatch(match.Headers)
	if err != nil {
		return nil, err
	}
	result := &apiv3_route.RouteMatch{
		Headers: headers,
	}

	switch match.Path.Type {
	case gw.PathMatchExact:
		result.PathSpecifier = &apiv3_route.RouteMatch_Path{Path: match.Path.Value}
	case gw.PathMatchPrefix:
		result.PathSpecifier = &apiv3_route.RouteMatch_Prefix{Prefix: match.Path.Value}
	case gw.PathMatchRegularExpression:
		result.PathSpecifier = &apiv3_route.RouteMatch_SafeRegex{SafeRegex: regexMatcher(match.Path.Value)}
	case "":
		// no path match, but PathSpecifier is required
		result.PathSpecifier = &apiv3_route.RouteMatch_Prefix{}
	default:
		return nil, errors.Errorf("unknown path match type: %q", match.Path.Type)
	}

	return result, nil
}

func Compile_HTTPHeaderMatch(headerMatch *gw.HTTPHeaderMatch) ([]*apiv3_route.HeaderMatcher, error) {
	if headerMatch == nil {
		return nil, nil
	}

	var result []*apiv3_route.HeaderMatcher
	for hdr, pattern := range headerMatch.Values {
		hm := &apiv3_route.HeaderMatcher{
			Name:        hdr,
			InvertMatch: false,
		}

		switch headerMatch.Type {
		case gw.HeaderMatchExact:
			hm.HeaderMatchSpecifier = &apiv3_route.HeaderMatcher_ExactMatch{ExactMatch: pattern}
		case gw.HeaderMatchRegularExpression:
			hm.HeaderMatchSpecifier = &apiv3_route.HeaderMatcher_SafeRegexMatch{SafeRegexMatch: regexMatcher(pattern)}
		default:
			return nil, errors.Errorf("unknown header match type: %s", headerMatch.Type)
		}

		result = append(result, hm)
	}
	return result, nil
}

func regexMatcher(pattern string) *apiv3_matcher.RegexMatcher {
	return &apiv3_matcher.RegexMatcher{
		EngineType: &apiv3_matcher.RegexMatcher_GoogleRe2{GoogleRe2: &apiv3_matcher.RegexMatcher_GoogleRE2{}},
		Regex:      pattern,
	}
}

func getName(resource kates.Object) string {
	return fmt.Sprintf("%s-%s", resource.GetNamespace(), resource.GetName())
}
