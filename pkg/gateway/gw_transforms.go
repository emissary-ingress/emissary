package gateway

import (
	"fmt"

	v2 "github.com/datawire/ambassador/v2/pkg/api/envoy/api/v2"
	core "github.com/datawire/ambassador/v2/pkg/api/envoy/api/v2/core"
	listener "github.com/datawire/ambassador/v2/pkg/api/envoy/api/v2/listener"
	route "github.com/datawire/ambassador/v2/pkg/api/envoy/api/v2/route"
	http "github.com/datawire/ambassador/v2/pkg/api/envoy/config/filter/network/http_connection_manager/v2"
	matcher "github.com/datawire/ambassador/v2/pkg/api/envoy/type/matcher"
	"github.com/datawire/ambassador/v2/pkg/envoy-control-plane/wellknown"
	"github.com/datawire/ambassador/v2/pkg/kates"
	"github.com/golang/protobuf/ptypes"
	"github.com/golang/protobuf/ptypes/wrappers"
	"github.com/pkg/errors"
	gw "sigs.k8s.io/gateway-api/apis/v1alpha1"
)

func Compile_Gateway(gateway *gw.Gateway) *CompiledConfig {
	src := SourceFromResource(gateway)
	var listeners []*CompiledListener
	for idx, l := range gateway.Spec.Listeners {
		name := fmt.Sprintf("%s-%d", getName(gateway), idx)
		listeners = append(listeners, Compile_Listener(src, l, name))
	}
	return &CompiledConfig{
		CompiledItem: NewCompiledItem(src),
		Listeners:    listeners,
	}
}

func Compile_Listener(parent Source, lst gw.Listener, name string) *CompiledListener {
	hcm := &http.HttpConnectionManager{
		StatPrefix: name,
		HttpFilters: []*http.HttpFilter{
			{Name: wellknown.CORS},
			{Name: wellknown.Router},
		},
		RouteSpecifier: &http.HttpConnectionManager_Rds{
			Rds: &http.Rds{
				ConfigSource: &core.ConfigSource{
					ConfigSourceSpecifier: &core.ConfigSource_Ads{
						Ads: &core.AggregatedConfigSource{},
					},
				},
				RouteConfigName: name,
			},
		},
	}
	hcmAny, err := ptypes.MarshalAny(hcm)
	if err != nil {
		panic(err)
	}

	return &CompiledListener{
		CompiledItem: NewCompiledItem(Sourcef("listener %s", lst.Hostname)),
		Listener: &v2.Listener{
			Name: name,
			Address: &core.Address{Address: &core.Address_SocketAddress{SocketAddress: &core.SocketAddress{
				Address:       "0.0.0.0",
				PortSpecifier: &core.SocketAddress_PortValue{PortValue: uint32(lst.Port)},
			}}},
			FilterChains: []*listener.FilterChain{
				{
					Filters: []*listener.Filter{
						{
							Name:       wellknown.HTTPConnectionManager,
							ConfigType: &listener.Filter_TypedConfig{TypedConfig: hcmAny},
						},
					},
				},
			},
		},
		Predicate: func(route *CompiledRoute) bool {
			return true
		},
		Domains: []string{"*"},
	}

}

func Compile_HTTPRoute(httpRoute *gw.HTTPRoute) *CompiledConfig {
	src := SourceFromResource(httpRoute)
	clusterRefs := []*ClusterRef{}
	var routes []*route.Route
	for idx, rule := range httpRoute.Spec.Rules {
		s := Sourcef("rule %d in %s", idx, src)
		routes = append(routes, Compile_HTTPRouteRule(s, rule, httpRoute.Namespace, &clusterRefs)...)
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
	}
}

func Compile_HTTPRouteRule(src Source, rule gw.HTTPRouteRule, namespace string, clusterRefs *[]*ClusterRef) (result []*route.Route) {
	var clusters []*route.WeightedCluster_ClusterWeight
	for idx, fwd := range rule.ForwardTo {
		s := Sourcef("forwardTo %d in %s", idx, src)
		clusters = append(clusters, Compile_HTTPRouteForwardTo(s, fwd, namespace, clusterRefs))
	}

	wc := &route.WeightedCluster{Clusters: clusters}

	for _, match := range Compile_HTTPRouteMatches(rule.Matches) {
		result = append(result, &route.Route{
			Match: match,
			Action: &route.Route_Route{Route: &route.RouteAction{
				ClusterSpecifier: &route.RouteAction_WeightedClusters{WeightedClusters: wc},
			}},
		})
	}

	return
}

func Compile_HTTPRouteForwardTo(src Source, forward gw.HTTPRouteForwardTo, namespace string, clusterRefs *[]*ClusterRef) *route.WeightedCluster_ClusterWeight {
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
	return &route.WeightedCluster_ClusterWeight{
		Name:   clusterName,
		Weight: &wrappers.UInt32Value{Value: uint32(forward.Weight)},
	}
}

func Compile_HTTPRouteMatches(matches []gw.HTTPRouteMatch) (result []*route.RouteMatch) {
	for _, match := range matches {
		result = append(result, Compile_HTTPRouteMatch(match))
	}
	return
}

func Compile_HTTPRouteMatch(match gw.HTTPRouteMatch) *route.RouteMatch {
	result := &route.RouteMatch{
		Headers: Compile_HTTPHeaderMatch(match.Headers),
	}

	switch match.Path.Type {
	case gw.PathMatchExact:
		result.PathSpecifier = &route.RouteMatch_Path{Path: match.Path.Value}
	case gw.PathMatchPrefix:
		result.PathSpecifier = &route.RouteMatch_Prefix{Prefix: match.Path.Value}
	case gw.PathMatchRegularExpression:
		result.PathSpecifier = &route.RouteMatch_SafeRegex{SafeRegex: regexMatcher(match.Path.Value)}
	case "":
		// no path match, but PathSpecifier is required
		result.PathSpecifier = &route.RouteMatch_Prefix{}
	default:
		panic(errors.Errorf("unknown path match type: %q", match.Path.Type))
	}

	return result
}

func Compile_HTTPHeaderMatch(headerMatch *gw.HTTPHeaderMatch) (result []*route.HeaderMatcher) {
	if headerMatch == nil {
		return
	}

	for hdr, pattern := range headerMatch.Values {
		hm := &route.HeaderMatcher{
			Name:        hdr,
			InvertMatch: false,
		}

		switch headerMatch.Type {
		case gw.HeaderMatchExact:
			hm.HeaderMatchSpecifier = &route.HeaderMatcher_ExactMatch{ExactMatch: pattern}
		case gw.HeaderMatchRegularExpression:
			hm.HeaderMatchSpecifier = &route.HeaderMatcher_SafeRegexMatch{SafeRegexMatch: regexMatcher(pattern)}
		default:
			panic(errors.Errorf("unknown header match type: %s", headerMatch.Type))
		}

		result = append(result, hm)
	}
	return
}

func regexMatcher(pattern string) *matcher.RegexMatcher {
	return &matcher.RegexMatcher{
		EngineType: &matcher.RegexMatcher_GoogleRe2{GoogleRe2: &matcher.RegexMatcher_GoogleRE2{}},
		Regex:      pattern,
	}
}

func getName(resource kates.Object) string {
	return fmt.Sprintf("%s-%s", resource.GetNamespace(), resource.GetName())
}
