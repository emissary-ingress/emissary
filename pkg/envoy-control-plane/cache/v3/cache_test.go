package cache_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"

	route "github.com/emissary-ingress/emissary/v3/pkg/api/envoy/config/route/v3"
	discovery "github.com/emissary-ingress/emissary/v3/pkg/api/envoy/service/discovery/v3"
	"github.com/emissary-ingress/emissary/v3/pkg/envoy-control-plane/cache/types"
	"github.com/emissary-ingress/emissary/v3/pkg/envoy-control-plane/cache/v3"
	"github.com/emissary-ingress/emissary/v3/pkg/envoy-control-plane/resource/v3"
)

const (
	resourceName = "route1"
)

func TestResponseGetDiscoveryResponse(t *testing.T) {
	routes := []types.ResourceWithTTL{{Resource: &route.RouteConfiguration{Name: resourceName}}}
	resp := cache.RawResponse{
		Request:   &discovery.DiscoveryRequest{TypeUrl: resource.RouteType},
		Version:   "v",
		Resources: routes,
	}

	discoveryResponse, err := resp.GetDiscoveryResponse()
	require.NoError(t, err)
	assert.Equal(t, discoveryResponse.GetVersionInfo(), resp.Version)
	assert.Len(t, discoveryResponse.GetResources(), 1)

	cachedResponse, err := resp.GetDiscoveryResponse()
	require.NoError(t, err)
	assert.Same(t, discoveryResponse, cachedResponse)

	r := &route.RouteConfiguration{}
	err = anypb.UnmarshalTo(discoveryResponse.GetResources()[0], r, proto.UnmarshalOptions{})
	require.NoError(t, err)
	assert.Equal(t, resourceName, r.GetName())
}

func TestPassthroughResponseGetDiscoveryResponse(t *testing.T) {
	routes := []types.Resource{&route.RouteConfiguration{Name: resourceName}}
	rsrc, err := anypb.New(routes[0])
	require.NoError(t, err)
	dr := &discovery.DiscoveryResponse{
		TypeUrl:     resource.RouteType,
		Resources:   []*anypb.Any{rsrc},
		VersionInfo: "v",
	}
	resp := cache.PassthroughResponse{
		Request:           &discovery.DiscoveryRequest{TypeUrl: resource.RouteType},
		DiscoveryResponse: dr,
	}

	discoveryResponse, err := resp.GetDiscoveryResponse()
	require.NoError(t, err)
	assert.Equal(t, "v", discoveryResponse.GetVersionInfo())
	assert.Len(t, discoveryResponse.GetResources(), 1)

	r := &route.RouteConfiguration{}
	err = anypb.UnmarshalTo(discoveryResponse.GetResources()[0], r, proto.UnmarshalOptions{})
	require.NoError(t, err)
	assert.Equal(t, resourceName, r.GetName())
	assert.Equal(t, discoveryResponse, dr)
}

func TestHeartbeatResponseGetDiscoveryResponse(t *testing.T) {
	routes := []types.ResourceWithTTL{{Resource: &route.RouteConfiguration{Name: resourceName}}}
	resp := cache.RawResponse{
		Request:   &discovery.DiscoveryRequest{TypeUrl: resource.RouteType},
		Version:   "v",
		Resources: routes,
		Heartbeat: true,
	}

	discoveryResponse, err := resp.GetDiscoveryResponse()
	require.NoError(t, err)
	assert.Equal(t, discoveryResponse.GetVersionInfo(), resp.Version)
	require.Len(t, discoveryResponse.GetResources(), 1)
	assert.False(t, isTTLResource(discoveryResponse.GetResources()[0]))

	cachedResponse, err := resp.GetDiscoveryResponse()
	require.NoError(t, err)
	assert.Same(t, discoveryResponse, cachedResponse)

	r := &route.RouteConfiguration{}
	err = anypb.UnmarshalTo(discoveryResponse.GetResources()[0], r, proto.UnmarshalOptions{})
	require.NoError(t, err)
	assert.Equal(t, resourceName, r.GetName())
}

func isTTLResource(resource *anypb.Any) bool {
	wrappedResource := &discovery.Resource{}
	err := protojson.Unmarshal(resource.GetValue(), wrappedResource)
	if err != nil {
		return false
	}

	return wrappedResource.GetResource() == nil
}
