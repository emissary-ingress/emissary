package main

import (
	"context"
	"log"
	"net"

	"github.com/gogo/googleapis/google/rpc"
	"google.golang.org/grpc"

	envoyCoreV2 "github.com/datawire/ambassador/go/apis/envoy/api/v2/core"
	envoyAuthV2 "github.com/datawire/ambassador/go/apis/envoy/service/auth/v2"
	envoyAuthV2alpha "github.com/datawire/ambassador/go/apis/envoy/service/auth/v2alpha"
)

func main() {
	socket, err := net.Listen("tcp", ":3000") // #nosec G102
	if err != nil {
		log.Fatal(err)
	}

	grpcServer := grpc.NewServer()
	envoyAuthV2alpha.RegisterAuthorizationServer(grpcServer, &AuthService{})
	envoyAuthV2.RegisterAuthorizationServer(grpcServer, &AuthService{})

	log.Print("starting...")
	log.Fatal(grpcServer.Serve(socket))
}

type AuthService struct{}

func (s *AuthService) Check(ctx context.Context, req *envoyAuthV2.CheckRequest) (*envoyAuthV2.CheckResponse, error) {
	return &envoyAuthV2.CheckResponse{
		Status: &rpc.Status{Code: int32(rpc.OK)},
		HttpResponse: &envoyAuthV2.CheckResponse_OkResponse{
			OkResponse: &envoyAuthV2.OkHttpResponse{
				Headers: []*envoyCoreV2.HeaderValueOption{},
			},
		},
	}, nil
}
