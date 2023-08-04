package cache

import (
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/emissary-ingress/emissary/v3/pkg/envoy-control-plane/resource/v3"
)

func TestOrderKeys(t *testing.T) {
	unorderedKeys := keys{
		{
			ID:      1,
			TypeURL: resource.EndpointType,
		},
		{
			ID:      2,
			TypeURL: resource.ClusterType,
		},
		{
			ID:      4,
			TypeURL: resource.ListenerType,
		},
		{
			ID:      3,
			TypeURL: resource.RouteType,
		},
		{
			ID:      5,
			TypeURL: resource.ScopedRouteType,
		},
	}
	expected := keys{
		{
			ID:      4,
			TypeURL: resource.ListenerType,
		},
		{
			ID:      3,
			TypeURL: resource.RouteType,
		},
		{
			ID:      5,
			TypeURL: resource.ScopedRouteType,
		},
		{
			ID:      2,
			TypeURL: resource.ClusterType,
		},
		{
			ID:      1,
			TypeURL: resource.EndpointType,
		},
	}

	orderedKeys := unorderedKeys
	sort.Sort(orderedKeys)

	assert.True(t, sort.IsSorted(orderedKeys))
	assert.NotEqual(t, unorderedKeys, &orderedKeys)
	assert.Equal(t, expected, orderedKeys)

	// Ordering:
	// === RUN   TestOrderKeys
	// order_test.go:43: {ID:4 TypeURL:type.googleapis.com/envoy.config.listener.v3.Listener}
	// order_test.go:43: {ID:3 TypeURL:type.googleapis.com/envoy.config.route.v3.RouteConfiguration}
	// order_test.go:43: {ID:5 TypeURL:type.googleapis.com/envoy.config.route.v3.ScopedRouteConfiguration}
	// order_test.go:43: {ID:2 TypeURL:type.googleapis.com/envoy.config.cluster.v3.Cluster}
	// order_test.go:43: {ID:1 TypeURL:type.googleapis.com/envoy.config.endpoint.v3.ClusterLoadAssignment}
}
