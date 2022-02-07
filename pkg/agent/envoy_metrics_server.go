package agent

import (
	"context"
	envoyMetrics "github.com/datawire/ambassador/v2/pkg/api/envoy/service/metrics/v3"
	"github.com/datawire/dlib/dhttp"
	"github.com/datawire/dlib/dlog"
	"google.golang.org/grpc"
	"io"
)

type streamHandler func(logCtx context.Context, in *envoyMetrics.StreamMetricsMessage)

type metricsServer struct {
	envoyMetrics.MetricsServiceServer
	handler streamHandler
}

// NewMetricsServer is the main metricsServer constructor.
func NewMetricsServer(handler streamHandler) *metricsServer {
	return &metricsServer{
		handler: handler,
	}
}

// StartServer will start the metrics gRPC server, listening on :8006
// It is a blocking call until sc.ListenAndServe returns.
func (s *metricsServer) StartServer(ctx context.Context) error {
	grpcServer := grpc.NewServer()
	envoyMetrics.RegisterMetricsServiceServer(grpcServer, s)

	sc := &dhttp.ServerConfig{
		Handler: grpcServer,
	}

	dlog.Info(ctx, "starting metrics service listening on :8006")
	return sc.ListenAndServe(ctx, ":8006")
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
