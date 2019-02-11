package services

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"

	core "github.com/datawire/ambassador/kat/backend/xds/envoy/api/v2/core"
	pb "github.com/datawire/ambassador/kat/backend/xds/envoy/service/auth/v2alpha"
	envoy_type "github.com/datawire/ambassador/kat/backend/xds/envoy/type"

	"github.com/gogo/googleapis/google/rpc"
	gogo_type "github.com/gogo/protobuf/types"
	"google.golang.org/grpc"
)

// GRPCAUTH server object (all fields are required).
type GRPCAUTH struct {
	Port       int16
	SecurePort int16
	Cert       string
	Key        string
}

// Start initializes the HTTP server.
func (g *GRPCAUTH) Start() <-chan bool {
	exited := make(chan bool)
	proto := "tcp"

	go func() {
		port := fmt.Sprintf(":%v", g.Port)

		ln, err := net.Listen(proto, port)
		if err != nil {
			log.Fatal()
		}

		s := grpc.NewServer()
		pb.RegisterAuthorizationServer(s, &AuthService{})
		s.Serve(ln)

		defer ln.Close()
		close(exited)
	}()

	go func() {
		cer, err := tls.LoadX509KeyPair(g.Cert, g.Key)
		if err != nil {
			log.Fatal(err)
			return
		}

		config := &tls.Config{Certificates: []tls.Certificate{cer}}
		port := fmt.Sprintf(":%v", g.SecurePort)
		ln, err := tls.Listen(proto, port, config)
		if err != nil {
			log.Fatal(err)
			return
		}

		s := grpc.NewServer()
		pb.RegisterAuthorizationServer(s, &AuthService{})
		s.Serve(ln)

		defer ln.Close()
		close(exited)
	}()

	log.Print("starting gRPC authorization service")
	return exited
}

// AuthService implements envoy.service.auth.external_auth.
type AuthService struct{}

// Check checks the request object.
func (s *AuthService) Check(ctx context.Context, r *pb.CheckRequest) (*pb.CheckResponse, error) {
	rs := &Response{}

	// These are the client request info that will go in the response body.
	request := r.GetAttributes().GetRequest().GetHttp().GetHeaders()
	request["body"] = r.GetAttributes().GetRequest().GetHttp().GetBody().String()

	// Sets requested HTTP status.
	rs.SetStatus(request["requested-status"])

	// Sets requested headers.
	for _, key := range strings.Split(request["requested-headers"], ",") {
		rs.AddHeader(true, key, request[key])
	}

	// Sets requested location.
	rs.AddHeader(true, "Location", request["requested-location"])

	// Write out all request/response information.
	response := make(map[string]interface{})
	response["headers"] = rs.GetHTTPHeaderMap()

	expected := make(map[string]interface{})
	expected["backend"] = os.Getenv("BACKEND")
	expected["request"] = request
	expected["response"] = response

	body, err := json.MarshalIndent(expected, "", "  ")
	if err != nil {
		body = []byte(fmt.Sprintf("Error: %v", err))
	}

	// Sets response body.
	rs.SetBody(string(body))

	log.Printf("writing response HTTP %v", request["requested-status"])
	return rs.GetResponse(), nil
}

// Response constructs an authorization response object.
type Response struct {
	headers []*core.HeaderValueOption
	body    string
	status  uint32
}

// AddHeader adds a header to the response. When append param is true, Envoy will
// append the value to an existent request header instead of overriding it.
func (r *Response) AddHeader(a bool, key, value string) {
	val := &core.HeaderValueOption{
		Header: &core.HeaderValue{
			Key:   key,
			Value: value,
		},
		Append: &gogo_type.BoolValue{Value: a},
	}
	r.headers = append(r.headers, val)
}

// // AddHeader adds a header option to the response.
// func (r *Response) AddHeader(header *core.HeaderValueOption) {
// 	r.headers = append(r.headers, header)
// }

// GetHTTPHeaderMap returns HTTP header mapping of the response header-options.
func (r *Response) GetHTTPHeaderMap() *http.Header {
	h := &http.Header{}
	for _, v := range r.headers {
		h.Add(v.Header.Key, v.Header.Value)
	}
	return h
}

// SetBody sets the authorization response message body.
func (r *Response) SetBody(s string) {
	r.body = s
}

// SetStatus sets the authorization response HTTP status code.
func (r *Response) SetStatus(s string) {
	if val, err := strconv.ParseUint(s, 10, 64); err != nil {
		r.status = uint32(val)
	} else {
		log.Print(err)
		r.status = uint32(500)
	}
}

// GetResponse returns the gRPC authorization response object.
func (r *Response) GetResponse() *pb.CheckResponse {
	rs := &pb.CheckResponse{}
	switch {
	// Ok respose.
	case r.status == 200 || r.status == 0:
		rs.Status = &rpc.Status{Code: int32(0)}
		rs.HttpResponse = &pb.CheckResponse_OkResponse{
			OkResponse: &pb.OkHttpResponse{
				Headers: r.headers,
			},
		}

	// Denied response.
	default:
		rs.Status = &rpc.Status{Code: int32(16)}
		rs.HttpResponse = &pb.CheckResponse_DeniedResponse{
			DeniedResponse: &pb.DeniedHttpResponse{
				Status: &envoy_type.HttpStatus{
					Code: envoy_type.StatusCode(r.status),
				},
				Headers: r.headers,
				Body:    r.body,
			},
		}
	}

	return rs
}
