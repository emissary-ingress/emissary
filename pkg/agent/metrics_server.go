package agent

import (
	"context"
	envoyMetrics "github.com/datawire/ambassador/v2/pkg/api/envoy/service/metrics/v2"
	"github.com/datawire/dlib/dlog"
	"google.golang.org/grpc"
	"io"
	"net"
)

type streamHandler func(in *envoyMetrics.StreamMetricsMessage)

type metricsServer struct {
	envoyMetrics.MetricsServiceServer
	handler streamHandler
}

// TODO(alexgervais): document
func NewMetricsServer(handler streamHandler) *metricsServer {
	return &metricsServer{
		handler: handler,
	}
}

// TODO(alexgervais): document
func (s *metricsServer) StartServer(ctx context.Context) error {
	grpcServer := grpc.NewServer()
	envoyMetrics.RegisterMetricsServiceServer(grpcServer, s)

	listener, err := net.Listen("tcp", ":8123")
	if err != nil {
		dlog.Errorf(ctx, "metrics service failed to listen: %v", err)
	}

	dlog.Infof(ctx, "metrics service listening on %s", listener.Addr().String())
	return grpcServer.Serve(listener)
}

// TODO(alexgervais): document
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
		s.handler(in)
	}
}
