package ttl

import (
	discovery "github.com/emissary-ingress/emissary/v3/pkg/api/envoy/service/discovery/v3"
	"github.com/emissary-ingress/emissary/v3/pkg/envoy-control-plane/cache/types"
	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/ptypes"
	"github.com/golang/protobuf/ptypes/any"
)

var deltaResourceTypeURL = "type.googleapis.com/" + proto.MessageName(&discovery.Resource{})

// Helper functions for interacting with TTL resources for xDS V3. A resource will be wrapped in a discovery.Resource in order
// to allow specifying a TTL. If the resource is meant to be a heartbeat response, only the resource name and TTL will be set
// to avoid having to send the entire resource down.

func MaybeCreateTtlResourceIfSupported(resource types.ResourceWithTtl, name string, resourceTypeUrl string, heartbeat bool) (types.Resource, string, error) {
	if resource.Ttl != nil {
		wrappedResource := &discovery.Resource{
			Name: name,
			Ttl:  ptypes.DurationProto(*resource.Ttl),
		}

		if !heartbeat {
			any, err := ptypes.MarshalAny(resource.Resource)
			if err != nil {
				return nil, "", err
			}
			any.TypeUrl = resourceTypeUrl
			wrappedResource.Resource = any
		}

		return wrappedResource, deltaResourceTypeURL, nil
	}

	return resource.Resource, resourceTypeUrl, nil
}

func IsTTLResource(resource *any.Any) bool {
	// This is only done in test, so no need to worry about the overhead of the marshalling.
	wrappedResource := &discovery.Resource{}
	err := ptypes.UnmarshalAny(resource, wrappedResource)
	if err != nil {
		return false
	}

	return wrappedResource.Resource == nil
}
