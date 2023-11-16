package ambex

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/wrapperspb"

	v3Listener "github.com/emissary-ingress/emissary/v3/pkg/api/envoy/config/listener/v3"
	v3Route "github.com/emissary-ingress/emissary/v3/pkg/api/envoy/config/route/v3"
	v3Httpman "github.com/emissary-ingress/emissary/v3/pkg/api/envoy/extensions/filters/network/http_connection_manager/v3"
	v3Wellknown "github.com/emissary-ingress/emissary/v3/pkg/envoy-control-plane/wellknown"
)

func TestV3ListenerToRdsListener(t *testing.T) {
	testRoute := &v3Route.Route_Route{
		Route: &v3Route.RouteAction{
			ClusterSpecifier: &v3Route.RouteAction_Cluster{
				Cluster: "cluster_quote_default_default",
			},
			PrefixRewrite: "/",
			Timeout:       durationpb.New(3 * time.Second),
		},
	}

	testHcm := &v3Httpman.HttpConnectionManager{
		RouteSpecifier: &v3Httpman.HttpConnectionManager_RouteConfig{
			RouteConfig: &v3Route.RouteConfiguration{
				VirtualHosts: []*v3Route.VirtualHost{{
					Name:    "emissary-ingress-listener-8080-*",
					Domains: []string{"*"},
					Routes: []*v3Route.Route{{
						Match: &v3Route.RouteMatch{
							PathSpecifier: &v3Route.RouteMatch_Prefix{
								Prefix: "/backend/",
							},
						},
						Action: testRoute,
					}},
				}},
			},
		},
	}

	anyTestHcm, err := anypb.New(testHcm)
	require.NoError(t, err)

	//Create a second identical Hcm
	anyTestHcm2, err := anypb.New(testHcm)
	require.NoError(t, err)

	testListener := &v3Listener.Listener{
		Name: "emissary-ingress-listener-8080",
		FilterChains: []*v3Listener.FilterChain{{
			Filters: []*v3Listener.Filter{{
				Name:       v3Wellknown.HTTPConnectionManager,
				ConfigType: &v3Listener.Filter_TypedConfig{TypedConfig: anyTestHcm},
			}, {
				Name:       v3Wellknown.HTTPConnectionManager,
				ConfigType: &v3Listener.Filter_TypedConfig{TypedConfig: anyTestHcm2},
			}},
			FilterChainMatch: &v3Listener.FilterChainMatch{
				DestinationPort: &wrapperspb.UInt32Value{Value: uint32(8080)},
			},
		}},
	}

	_, routes, err := V3ListenerToRdsListener(testListener)
	require.NoError(t, err)

	//Should have 2 routes
	assert.Equal(t, 2, len(routes))

	for i, rc := range routes {
		// Confirm that the route name was transformed to the hashed version
		assert.Equal(t, fmt.Sprintf("emissary-ingress-listener-8080-routeconfig-8c82e45fa3f94ab4e879543e0a1a30ac-%d", i), rc.GetName())

		// Make sure the virtual hosts are unmodified
		virtualHosts := rc.GetVirtualHosts()
		assert.Equal(t, 1, len(virtualHosts))
		assert.Equal(t, "emissary-ingress-listener-8080-*", virtualHosts[0].GetName())
		assert.Equal(t, []string{"*"}, virtualHosts[0].GetDomains())
	}
}
