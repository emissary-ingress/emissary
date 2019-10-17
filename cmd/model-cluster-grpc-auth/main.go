package main

import (
	"context"
	"log"
	"net"
	"net/http"
	"net/url"

	"github.com/gogo/protobuf/types"
	"google.golang.org/grpc"
	rpc "istio.io/gogo-genproto/googleapis/google/rpc"

	envoyCoreV2 "github.com/datawire/ambassador/pkg/api/envoy/api/v2/core"
	envoyAuthV2 "github.com/datawire/ambassador/pkg/api/envoy/service/auth/v2"
	envoyAuthV2alpha "github.com/datawire/ambassador/pkg/api/envoy/service/auth/v2alpha"
	envoyType "github.com/datawire/ambassador/pkg/api/envoy/type"
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
	log.Println("ACCESS",
		req.GetAttributes().GetRequest().GetHttp().GetMethod(),
		req.GetAttributes().GetRequest().GetHttp().GetHost(),
		req.GetAttributes().GetRequest().GetHttp().GetPath(),
	)
	requestURI, err := url.ParseRequestURI(req.GetAttributes().GetRequest().GetHttp().GetPath())
	if err != nil {
		log.Println("=> ERROR", err)
		return &envoyAuthV2.CheckResponse{
			Status: &rpc.Status{Code: int32(rpc.UNKNOWN)},
			HttpResponse: &envoyAuthV2.CheckResponse_DeniedResponse{
				DeniedResponse: &envoyAuthV2.DeniedHttpResponse{
					Status: &envoyType.HttpStatus{Code: http.StatusInternalServerError},
					Headers: []*envoyCoreV2.HeaderValueOption{
						{Header: &envoyCoreV2.HeaderValue{Key: "Content-Type", Value: "application/json"}},
					},
					Body: `{"msg": "internal server error"}`,
				},
			},
		}, nil
	}
	switch requestURI.Path {
	case "/external-grpc/headers":
		log.Print("=> ALLOW")
		header := make([]*envoyCoreV2.HeaderValueOption, 0, 4+len(req.GetAttributes().GetRequest().GetHttp().GetHeaders()))
		for k, v := range req.GetAttributes().GetRequest().GetHttp().GetHeaders() {
			header = append(header, &envoyCoreV2.HeaderValueOption{
				Header: &envoyCoreV2.HeaderValue{Key: "X-Input-" + k, Value: v},
				Append: &types.BoolValue{Value: false},
			})
		}
		header = append(header, &envoyCoreV2.HeaderValueOption{
			Header: &envoyCoreV2.HeaderValue{Key: "X-Allowed-Input-Header", Value: "after"},
			Append: &types.BoolValue{Value: true},
		})
		header = append(header, &envoyCoreV2.HeaderValueOption{
			Header: &envoyCoreV2.HeaderValue{Key: "X-Disallowed-Input-Header", Value: "after"},
			Append: &types.BoolValue{Value: false},
		})
		header = append(header, &envoyCoreV2.HeaderValueOption{
			Header: &envoyCoreV2.HeaderValue{Key: "X-Allowed-Output-Header", Value: "baz"},
			Append: &types.BoolValue{Value: true},
		})
		header = append(header, &envoyCoreV2.HeaderValueOption{
			Header: &envoyCoreV2.HeaderValue{Key: "X-Disallowed-Output-Header", Value: "qux"},
			Append: &types.BoolValue{Value: false},
		})
		return &envoyAuthV2.CheckResponse{
			Status: &rpc.Status{Code: int32(rpc.OK)},
			HttpResponse: &envoyAuthV2.CheckResponse_OkResponse{
				OkResponse: &envoyAuthV2.OkHttpResponse{
					Headers: header,
				},
			},
		}, nil
	default:
		log.Print("=> DENY")
		return &envoyAuthV2.CheckResponse{
			Status: &rpc.Status{Code: int32(rpc.UNKNOWN)},
			HttpResponse: &envoyAuthV2.CheckResponse_DeniedResponse{
				DeniedResponse: &envoyAuthV2.DeniedHttpResponse{
					Status: &envoyType.HttpStatus{Code: http.StatusOK},
					Headers: []*envoyCoreV2.HeaderValueOption{
						{Header: &envoyCoreV2.HeaderValue{Key: "X-Allowed-Output-Header", Value: "baz"}},
						{Header: &envoyCoreV2.HeaderValue{Key: "X-Disallowed-Output-Header", Value: "qux"}},
						{Header: &envoyCoreV2.HeaderValue{Key: "Content-Type", Value: "application/json"}},
					},
					Body: `{"msg": "intercepted"}`,
				},
			},
		}, nil
	}
}
