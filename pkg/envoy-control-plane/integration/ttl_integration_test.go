package integration

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"

	envoy_config_core_v3 "github.com/emissary-ingress/emissary/v3/pkg/api/envoy/config/core/v3"
	envoy_config_endpoint_v3 "github.com/emissary-ingress/emissary/v3/pkg/api/envoy/config/endpoint/v3"
	envoy_service_discovery_v3 "github.com/emissary-ingress/emissary/v3/pkg/api/envoy/service/discovery/v3"
	endpointservice "github.com/emissary-ingress/emissary/v3/pkg/api/envoy/service/endpoint/v3"
	"github.com/emissary-ingress/emissary/v3/pkg/envoy-control-plane/cache/types"
	"github.com/emissary-ingress/emissary/v3/pkg/envoy-control-plane/cache/v3"
	"github.com/emissary-ingress/emissary/v3/pkg/envoy-control-plane/resource/v3"
	"github.com/emissary-ingress/emissary/v3/pkg/envoy-control-plane/server/v3"
)

type logger struct {
	t *testing.T
}

func (log logger) Debugf(format string, args ...interface{}) { log.t.Logf(format, args...) }
func (log logger) Infof(format string, args ...interface{})  { log.t.Logf(format, args...) }
func (log logger) Warnf(format string, args ...interface{})  { log.t.Logf(format, args...) }
func (log logger) Errorf(format string, args ...interface{}) { log.t.Logf(format, args...) }

func TestTTLResponse(t *testing.T) {

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	snapshotCache := cache.NewSnapshotCacheWithHeartbeating(ctx, false, cache.IDHash{}, logger{t: t}, time.Second)

	server := server.NewServer(ctx, snapshotCache, nil)

	grpcServer := grpc.NewServer()
	endpointservice.RegisterEndpointDiscoveryServiceServer(grpcServer, server)

	l, err := net.Listen("tcp", ":9999") // nolint:gosec
	assert.NoError(t, err)

	go func() {
		assert.NoError(t, grpcServer.Serve(l))
	}()
	defer grpcServer.Stop()

	conn, err := grpc.Dial(":9999", grpc.WithTransportCredentials(insecure.NewCredentials()))
	assert.NoError(t, err)
	client := endpointservice.NewEndpointDiscoveryServiceClient(conn)

	sclient, err := client.StreamEndpoints(ctx)
	assert.NoError(t, err)

	err = sclient.Send(&envoy_service_discovery_v3.DiscoveryRequest{
		Node: &envoy_config_core_v3.Node{
			Id: "test",
		},
		ResourceNames: []string{"resource"},
		TypeUrl:       resource.EndpointType,
	})
	assert.NoError(t, err)

	oneSecond := time.Second
	cla := &envoy_config_endpoint_v3.ClusterLoadAssignment{ClusterName: "resource"}
	snap, _ := cache.NewSnapshotWithTTLs("1", map[resource.Type][]types.ResourceWithTTL{
		resource.EndpointType: {{
			Resource: cla,
			TTL:      &oneSecond,
		}},
	})
	err = snapshotCache.SetSnapshot(context.Background(), "test", snap)
	assert.NoError(t, err)

	timeout := time.NewTimer(5 * time.Second)

	awaitResponse := func() *envoy_service_discovery_v3.DiscoveryResponse {
		t.Helper()
		doneCh := make(chan *envoy_service_discovery_v3.DiscoveryResponse)
		go func() {

			r, err := sclient.Recv()
			assert.NoError(t, err)

			doneCh <- r
		}()

		select {
		case <-timeout.C:
			assert.Fail(t, "timed out")
			return nil
		case r := <-doneCh:
			return r
		}
	}

	response := awaitResponse()
	isFullResponseWithTTL(t, response)

	err = sclient.Send(&envoy_service_discovery_v3.DiscoveryRequest{
		Node: &envoy_config_core_v3.Node{
			Id: "test",
		},
		ResourceNames: []string{"resource"},
		TypeUrl:       resource.EndpointType,
		VersionInfo:   "1",
		ResponseNonce: response.Nonce,
	})
	assert.NoError(t, err)

	response = awaitResponse()
	isHeartbeatResponseWithTTL(t, response)
}

func isFullResponseWithTTL(t *testing.T, response *envoy_service_discovery_v3.DiscoveryResponse) {
	t.Helper()

	assert.Len(t, response.Resources, 1)
	r := response.Resources[0]
	resource := &envoy_service_discovery_v3.Resource{}
	err := anypb.UnmarshalTo(r, resource, proto.UnmarshalOptions{})
	assert.NoError(t, err)

	assert.NotNil(t, resource.Ttl)
	assert.NotNil(t, resource.Resource)
}

func isHeartbeatResponseWithTTL(t *testing.T, response *envoy_service_discovery_v3.DiscoveryResponse) {
	t.Helper()

	assert.Len(t, response.Resources, 1)
	r := response.Resources[0]
	resource := &envoy_service_discovery_v3.Resource{}
	err := anypb.UnmarshalTo(r, resource, proto.UnmarshalOptions{})
	assert.NoError(t, err)

	assert.NotNil(t, resource.Ttl)
	assert.Nil(t, resource.Resource)
}
