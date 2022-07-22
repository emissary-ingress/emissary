package main

import (
	"context"
	"io"
	"os"

	"github.com/datawire/dlib/dhttp"
	"github.com/datawire/dlib/dlog"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/encoding/protojson"

	v2_metrics "github.com/emissary-ingress/emissary/v3/pkg/api/envoy/service/metrics/v2"
)

type server struct{}

var _ v2_metrics.MetricsServiceServer = &server{}

func main() {
	ctx := context.Background()

	grpcMux := grpc.NewServer()
	v2_metrics.RegisterMetricsServiceServer(grpcMux, &server{})

	sc := &dhttp.ServerConfig{
		Handler: grpcMux,
	}

	dlog.Print(ctx, "starting...")

	if err := sc.ListenAndServe(ctx, ":8080"); err != nil {
		dlog.Errorf(ctx, "shut down with error: %v", err)
		os.Exit(1)
	}

	dlog.Print(ctx, "shut down without error")
}

func (s *server) StreamMetrics(stream v2_metrics.MetricsService_StreamMetricsServer) error {
	dlog.Println(stream.Context(), "Started stream")
	for {
		in, err := stream.Recv()

		if err == io.EOF {
			return nil
		}

		if err != nil {
			return err
		}

		dlog.Println(stream.Context(), protojson.Format(in))
	}
}
