package stream

import (
	"google.golang.org/grpc"

	discovery "github.com/datawire/ambassador/v2/pkg/api/envoy/service/discovery/v3"
)

// Generic RPC stream.
type Stream interface {
	grpc.ServerStream

	Send(*discovery.DiscoveryResponse) error
	Recv() (*discovery.DiscoveryRequest, error)
}

type DeltaStream interface {
	grpc.ServerStream

	Send(*discovery.DeltaDiscoveryResponse) error
	Recv() (*discovery.DeltaDiscoveryRequest, error)
}

// StreamState will keep track of resource state on a stream
type StreamState struct {
	// Indicates whether the original DeltaRequest was a wildcard LDS/RDS request.
	IsWildcard bool

	// ResourceVersions contains a hash of the resource as the value and the resource name as the key.
	// This field stores the last state sent to the client.
	ResourceVersions map[string]string
}
