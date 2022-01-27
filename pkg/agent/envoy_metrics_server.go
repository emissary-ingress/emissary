package agent

import (
	"context"
	envoyMetrics "github.com/datawire/ambassador/v2/pkg/api/envoy/service/metrics/v2"
	"github.com/datawire/dlib/dlog"
	"google.golang.org/grpc"
	"io"
	"net"
)

type streamHandler func(logCtx context.Context, in *envoyMetrics.StreamMetricsMessage)

type metricsServer struct {
	envoyMetrics.MetricsServiceServer
	handler streamHandler
	logCtx  context.Context
}

// NewMetricsServer is the main metricsServer constructor.
func NewMetricsServer(handler streamHandler) *metricsServer {
	return &metricsServer{
		handler: handler,
	}
}

// StartServer will start the metrics gRPC server, listening on :8123
// It is a blocking call until grpcServer.Serve returns.
func (s *metricsServer) StartServer(ctx context.Context) error {
	grpcServer := grpc.NewServer()
	envoyMetrics.RegisterMetricsServiceServer(grpcServer, s)

	listener, err := net.Listen("tcp", ":8123")
	if err != nil {
		dlog.Errorf(ctx, "metrics service failed to listen: %v", err)
	}

	dlog.Infof(ctx, "metrics service listening on %s", listener.Addr().String())
	s.logCtx = ctx
	return grpcServer.Serve(listener)
}

// StreamMetrics implements the StreamMetrics rpc call by calling the stream handler on each
// message received. It's invoked whenever metrics arrive from Envoy.
func (s *metricsServer) StreamMetrics(stream envoyMetrics.MetricsService_StreamMetricsServer) error {
	dlog.Debug(s.logCtx, "started stream")
	for {
		in, err := stream.Recv()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		s.handler(s.logCtx, in)
	}
}
