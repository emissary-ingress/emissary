package services

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"

	// "os"
	"strconv"
	"strings"

	core "github.com/datawire/ambassador/pkg/api/envoy/api/v2/core"
	pb "github.com/datawire/ambassador/pkg/api/envoy/service/auth/v2"
	pb_legacy "github.com/datawire/ambassador/pkg/api/envoy/service/auth/v2alpha"
	envoy_type "github.com/datawire/ambassador/pkg/api/envoy/type"

	gogo_type "github.com/gogo/protobuf/types"
	"google.golang.org/grpc"
	rpc "istio.io/gogo-genproto/googleapis/google/rpc"
)

// GRPCAUTH server object (all fields are required).
type GRPCAUTH struct {
	Port          int16
	Backend       string
	SecurePort    int16
	SecureBackend string
	Cert          string
	Key           string
}

// Start initializes the HTTP server.
func (g *GRPCAUTH) Start() <-chan bool {
	log.Printf("GRPCAUTH: %s listening on %d/%d", g.Backend, g.Port, g.SecurePort)

	exited := make(chan bool)
	proto := "tcp"

	go func() {
		port := fmt.Sprintf(":%v", g.Port)

		ln, err := net.Listen(proto, port)
		if err != nil {
			log.Fatal()
		}

		s := grpc.NewServer()
		pb_legacy.RegisterAuthorizationServer(s, g)
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
		pb_legacy.RegisterAuthorizationServer(s, g)
		s.Serve(ln)

		defer ln.Close()
		close(exited)
	}()

	log.Print("starting gRPC authorization service")
	return exited
}

// Check checks the request object.
func (g *GRPCAUTH) Check(ctx context.Context, r *pb.CheckRequest) (*pb.CheckResponse, error) {
	rs := &Response{}

	rheader := r.GetAttributes().GetRequest().GetHttp().GetHeaders()
	rbody := r.GetAttributes().GetRequest().GetHttp().GetBody()
	if len(rbody) > 0 {
		rheader["body"] = rbody
	}

	// Sets requested HTTP status.
	rs.SetStatus(rheader["requested-status"])

	// Sets requested headers.
	for _, key := range strings.Split(rheader["requested-header"], ",") {
		if val := rheader[key]; len(val) > 0 {
			rs.AddHeader(false, key, val)
		}
	}

	// Append requested headers.
	for _, token := range strings.Split(rheader["x-grpc-auth-append"], ";") {
		header := strings.Split(strings.TrimSpace(token), "=")
		if len(header) > 1 {
			log.Printf("appending header %s : %s", header[0], header[1])
			rs.AddHeader(true, header[0], header[1])
		}
	}

	// Sets requested Cookies.
	for _, v := range strings.Split(rheader["requested-cookie"], ",") {
		val := strings.Trim(v, " ")
		rs.AddHeader(false, "Set-Cookie", fmt.Sprintf("%s=%s", val, val))
	}

	// Sets requested location.
	if len(rheader["requested-location"]) > 0 {
		rs.AddHeader(false, "Location", rheader["requested-location"])
	}

	// Parses request headers.
	headers := make(map[string]interface{})
	for k, v := range rheader {
		headers[k] = strings.Split(v, ",")
	}

	// Parses request URL.
	url := make(map[string]interface{})
	url["fragment"] = r.GetAttributes().GetRequest().GetHttp().GetFragment()
	url["host"] = r.GetAttributes().GetRequest().GetHttp().GetHost()
	url["path"] = r.GetAttributes().GetRequest().GetHttp().GetPath()
	url["query"] = r.GetAttributes().GetRequest().GetHttp().GetQuery()
	url["scheme"] = r.GetAttributes().GetRequest().GetHttp().GetScheme()

	// Parses TLS info.
	tls := make(map[string]interface{})
	tls["enabled"] = false

	// Sets request portion of the results body.
	request := make(map[string]interface{})
	request["url"] = url
	request["method"] = r.GetAttributes().GetRequest().GetHttp().GetMethod()
	request["headers"] = headers
	request["host"] = r.GetAttributes().GetRequest().GetHttp().GetHost()
	request["tls"] = tls

	// Sets results body.
	results := make(map[string]interface{})
	results["backend"] = g.Backend
	results["status"] = rs.GetStatus()
	if len(request) > 0 {
		results["request"] = request
	}
	if rs.GetHTTPHeaderMap() != nil {
		results["headers"] = *rs.GetHTTPHeaderMap()
	}
	body, err := json.MarshalIndent(results, "", "  ")
	if err != nil {
		body = []byte(fmt.Sprintf("Error: %v", err))
	}

	// Sets response body.
	log.Printf("setting response body: %s", string(body))
	rs.SetBody(string(body))

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
func (r *Response) AddHeader(a bool, k, v string) {
	val := &core.HeaderValueOption{
		Header: &core.HeaderValue{
			Key:   k,
			Value: v,
		},
		Append: &gogo_type.BoolValue{Value: a},
	}
	r.headers = append(r.headers, val)
}

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
	if len(s) == 0 {
		s = "200"
	}
	if val, err := strconv.Atoi(s); err == nil {
		r.status = uint32(val)
		r.AddHeader(false, "status", s)
		log.Printf("setting HTTP status %v", r.status)
	} else {
		r.status = uint32(500)
		r.AddHeader(false, "status", "500")
		log.Printf("error setting HTTP status. Cannot parse string %s: %v.", s, err)
	}
}

// GetStatus returns the authorization response HTTP status code.
func (r *Response) GetStatus() uint32 {
	return r.status
}

// GetResponse returns the gRPC authorization response object.
func (r *Response) GetResponse() *pb.CheckResponse {
	rs := &pb.CheckResponse{}
	switch {
	// Ok respose.
	case r.status == http.StatusOK || r.status == 0:
		rs.Status = &rpc.Status{Code: int32(rpc.OK)}
		rs.HttpResponse = &pb.CheckResponse_OkResponse{
			OkResponse: &pb.OkHttpResponse{
				Headers: r.headers,
			},
		}

	// Denied response.
	default:
		rs.Status = &rpc.Status{Code: int32(rpc.UNAUTHENTICATED)}
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
