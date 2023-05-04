package sotw_test

import (
	"context"
	"net"
	"sync"
	"testing"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"

	clusterv3 "github.com/emissary-ingress/emissary/v3/pkg/api/envoy/config/cluster/v3"
	core "github.com/emissary-ingress/emissary/v3/pkg/api/envoy/config/core/v3"
	discovery "github.com/emissary-ingress/emissary/v3/pkg/api/envoy/service/discovery/v3"
	"github.com/emissary-ingress/emissary/v3/pkg/envoy-control-plane/cache/types"
	"github.com/emissary-ingress/emissary/v3/pkg/envoy-control-plane/cache/v3"
	client "github.com/emissary-ingress/emissary/v3/pkg/envoy-control-plane/client/sotw/v3"
	"github.com/emissary-ingress/emissary/v3/pkg/envoy-control-plane/resource/v3"
	"github.com/emissary-ingress/emissary/v3/pkg/envoy-control-plane/server/v3"

	"github.com/stretchr/testify/assert"
)

func TestFetch(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	snapCache := cache.NewSnapshotCache(true, cache.IDHash{}, nil)
	go func() {
		err := startAdsServer(ctx, snapCache)
		assert.NoError(t, err)
	}()

	conn, err := grpc.Dial(":18001", grpc.WithTransportCredentials(insecure.NewCredentials()), grpc.WithBlock())
	assert.NoError(t, err)
	defer conn.Close()

	c := client.NewADSClient(ctx, &core.Node{Id: "node_1"}, resource.ClusterType)
	err = c.InitConnect(conn)
	assert.NoError(t, err)

	t.Run("Test initial fetch", testInitialFetch(ctx, snapCache, c))
	t.Run("Test next fetch", testNextFetch(ctx, snapCache, c))
}

func testInitialFetch(ctx context.Context, snapCache cache.SnapshotCache, c client.ADSClient) func(t *testing.T) {
	return func(t *testing.T) {
		wg := sync.WaitGroup{}
		wg.Add(1)

		go func() {
			// watch for configs
			resp, err := c.Fetch()
			assert.NoError(t, err)
			assert.Equal(t, 3, len(resp.Resources))
			for _, r := range resp.Resources {
				cluster := &clusterv3.Cluster{}
				err := anypb.UnmarshalTo(r, cluster, proto.UnmarshalOptions{})
				assert.NoError(t, err)
				assert.Contains(t, []string{"cluster_1", "cluster_2", "cluster_3"}, cluster.Name)
			}

			err = c.Ack()
			assert.NoError(t, err)
			wg.Done()
		}()

		snapshot, err := cache.NewSnapshot("1", map[resource.Type][]types.Resource{
			resource.ClusterType: {
				&clusterv3.Cluster{Name: "cluster_1"},
				&clusterv3.Cluster{Name: "cluster_2"},
				&clusterv3.Cluster{Name: "cluster_3"},
			},
		})
		assert.NoError(t, err)

		err = snapshot.Consistent()
		assert.NoError(t, err)
		err = snapCache.SetSnapshot(ctx, "node_1", snapshot)
		wg.Wait()
		assert.NoError(t, err)
	}
}

func testNextFetch(ctx context.Context, snapCache cache.SnapshotCache, c client.ADSClient) func(t *testing.T) {
	return func(t *testing.T) {
		wg := sync.WaitGroup{}
		wg.Add(1)

		go func() {
			// watch for configs
			resp, err := c.Fetch()
			assert.NoError(t, err)
			assert.Equal(t, 2, len(resp.Resources))
			for _, r := range resp.Resources {
				cluster := &clusterv3.Cluster{}
				err = anypb.UnmarshalTo(r, cluster, proto.UnmarshalOptions{})
				assert.NoError(t, err)
				assert.Contains(t, []string{"cluster_2", "cluster_4"}, cluster.Name)
			}

			err = c.Ack()
			assert.NoError(t, err)
			wg.Done()
		}()

		snapshot, err := cache.NewSnapshot("2", map[resource.Type][]types.Resource{
			resource.ClusterType: {
				&clusterv3.Cluster{Name: "cluster_2"},
				&clusterv3.Cluster{Name: "cluster_4"},
			},
		})
		assert.NoError(t, err)

		err = snapshot.Consistent()
		assert.NoError(t, err)
		err = snapCache.SetSnapshot(ctx, "node_1", snapshot)
		assert.NoError(t, err)
		wg.Wait()
	}
}

func startAdsServer(ctx context.Context, snapCache cache.SnapshotCache) error {
	lis, err := net.Listen("tcp", "127.0.0.1:18001")
	if err != nil {
		return err
	}

	grpcServer := grpc.NewServer()
	s := server.NewServer(ctx, snapCache, nil)
	discovery.RegisterAggregatedDiscoveryServiceServer(grpcServer, s)

	if e := grpcServer.Serve(lis); e != nil {
		err = e
	}

	return err
}
