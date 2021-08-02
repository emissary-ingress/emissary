package main

import (
	"io"
	"log"
	"net"

	v2 "github.com/datawire/ambassador/v2/pkg/api/envoy/service/metrics/v2"
	"github.com/golang/protobuf/jsonpb"
	"google.golang.org/grpc"
)

func main() {
	grpcServer := grpc.NewServer()
	v2.RegisterMetricsServiceServer(grpcServer, New())

	l, err := net.Listen("tcp", ":8123")
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	log.Println("Listening on tcp://localhost:8123")
	grpcServer.Serve(l)
}

type server struct {
	marshaler jsonpb.Marshaler
}

var _ v2.MetricsServiceServer = &server{}

// New ...
func New() v2.MetricsServiceServer {
	return &server{
		marshaler: jsonpb.Marshaler{
			Indent: "  ",
		},
	}
}

func (s *server) StreamMetrics(stream v2.MetricsService_StreamMetricsServer) error {
	log.Println("Started stream")
	for {
		in, err := stream.Recv()
		log.Println("Received value")
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		str, _ := s.marshaler.MarshalToString(in)
		log.Println(str)
	}
}
