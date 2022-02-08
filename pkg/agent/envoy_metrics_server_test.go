package agent_test

import (
	"context"
	"errors"
	"fmt"
	"io"
	"testing"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/datawire/ambassador/v2/pkg/agent"
	envoyMetrics "github.com/datawire/ambassador/v2/pkg/api/envoy/service/metrics/v3"
	"github.com/datawire/dlib/dgroup"
	"github.com/datawire/dlib/dlog"
)

// TestMetricsContext checks that the parent Context correctly gets passed through to the metrics
// handler function.
//
// There was concern that without storing a "logCtx" in the metricsServer{} that .StreamMetrics
// would end up using the fallback logger.  This concern was based on the google.golang.org/grpc
// HTTP server not correctly passing Contexts through.  Since we insist on using the
// github.com/datawire/dlib/dhttp HTTP server instead, this isn't a problem; but let's add a test
// for it anyway, to put minds at ease.
func TestMetricsContext(t *testing.T) {
	grp := dgroup.NewGroup(dlog.NewTestContext(t, true), dgroup.GroupConfig{
		ShutdownOnNonError: true,
	})
	grp.Go("server", func(ctx context.Context) error {
		type testCtxKey struct{}
		ctx = context.WithValue(ctx, testCtxKey{}, "sentinel")
		srv := agent.NewMetricsServer(func(ctx context.Context, _ *envoyMetrics.StreamMetricsMessage) {
			if val, _ := ctx.Value(testCtxKey{}).(string); val != "sentinel" {
				t.Error("context did not get passed through")
			} else {
				t.Log("SUCCESS!!")
			}
		})
		return srv.StartServer(ctx)
	})
	grp.Go("client", func(ctx context.Context) error {
		grpcClient, err := grpc.DialContext(ctx, "localhost:8080",
			grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err != nil {
			return fmt.Errorf("grpc.DialContext: %w", err)
		}
		metricsClient := envoyMetrics.NewMetricsServiceClient(grpcClient)
		stream, err := metricsClient.StreamMetrics(ctx)
		if err != nil {
			return fmt.Errorf("metricsClient.StreamMetrics: %w", err)
		}
		if err := stream.Send(&envoyMetrics.StreamMetricsMessage{}); err != nil {
			return fmt.Errorf("stream.Send: %w", err)
		}
		if _, err := stream.CloseAndRecv(); err != nil && !errors.Is(err, io.EOF) {
			return fmt.Errorf("stream.CloseAndRecv: %w", err)
		}
		return nil
	})
	if err := grp.Wait(); err != nil {
		t.Error(err)
	}
}
