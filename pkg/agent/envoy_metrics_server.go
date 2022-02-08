package agent

import (
	"context"
	"io"

	"google.golang.org/grpc"

	envoyMetrics "github.com/datawire/ambassador/v2/pkg/api/envoy/service/metrics/v3"
	"github.com/datawire/dlib/dhttp"
	"github.com/datawire/dlib/dlog"
)

type StreamHandler func(ctx context.Context, in *envoyMetrics.StreamMetricsMessage)

type metricsServer struct {
	envoyMetrics.MetricsServiceServer
	handler StreamHandler
}

// NewMetricsServer is the main metricsServer constructor.
func NewMetricsServer(handler StreamHandler) *metricsServer {
	return &metricsServer{
		handler: handler,
	}
}

// StartServer will start the metrics gRPC server, listening on :8080
// It is a blocking call until sc.ListenAndServe returns.
func (s *metricsServer) StartServer(ctx context.Context) error {
	grpcServer := grpc.NewServer()
	envoyMetrics.RegisterMetricsServiceServer(grpcServer, s)

	sc := &dhttp.ServerConfig{
		Handler: grpcServer,
	}

	dlog.Info(ctx, "starting metrics service listening on :8080")
	return sc.ListenAndServe(ctx, ":8080")
}

// StreamMetrics implements the StreamMetrics rpc call by calling the stream handler on each
// message received. It's invoked whenever metrics arrive from Envoy.
func (s *metricsServer) StreamMetrics(stream envoyMetrics.MetricsService_StreamMetricsServer) error {
	ctx := stream.Context()
	dlog.Debug(ctx, "started stream")
	for {
		in, err := stream.Recv()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		s.handler(ctx, in)
	}
}
