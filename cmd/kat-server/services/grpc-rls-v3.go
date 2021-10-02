package services

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"strings"

	"github.com/golang/protobuf/ptypes/wrappers"
	"google.golang.org/grpc"

	core "github.com/datawire/ambassador/v2/pkg/api/envoy/config/core/v3"
	pb "github.com/datawire/ambassador/v2/pkg/api/envoy/service/ratelimit/v3"
	"github.com/datawire/dlib/dlog"
)

// GRPCRLSV3 server object (all fields are required).
type GRPCRLSV3 struct {
	Port            int16
	Backend         string
	SecurePort      int16
	SecureBackend   string
	Cert            string
	Key             string
	ProtocolVersion string
}

// Start initializes the HTTP server.
func (g *GRPCRLSV3) Start(ctx context.Context) <-chan bool {
	dlog.Printf(ctx, "GRPCRLSV3: %s listening on %d/%d", g.Backend, g.Port, g.SecurePort)

	exited := make(chan bool)
	proto := "tcp"

	go func() {
		port := fmt.Sprintf(":%v", g.Port)

		ln, err := net.Listen(proto, port)
		if err != nil {
			dlog.Error(ctx, err)
			panic(err) // TODO: do something better
		}

		s := grpc.NewServer()
		dlog.Printf(ctx, "registering v3 service")
		pb.RegisterRateLimitServiceServer(s, g)
		if err := s.Serve(ln); err != nil {
			panic(err) // TODO: do something better
		}

		defer ln.Close()
		close(exited)
	}()

	go func() {
		cer, err := tls.LoadX509KeyPair(g.Cert, g.Key)
		if err != nil {
			dlog.Error(ctx, err)
			panic(err) // TODO: do something better
		}

		config := &tls.Config{Certificates: []tls.Certificate{cer}}
		port := fmt.Sprintf(":%v", g.SecurePort)
		ln, err := tls.Listen(proto, port, config)
		if err != nil {
			dlog.Error(ctx, err)
			panic(err)
			// TODO: do something better
		}

		s := grpc.NewServer()
		dlog.Printf(ctx, "registering v3 service")
		pb.RegisterRateLimitServiceServer(s, g)
		if err := s.Serve(ln); err != nil {
			panic(err) // TODO: do something better
		}

		defer ln.Close()
		close(exited)
	}()

	dlog.Print(ctx, "starting gRPC rls service")
	return exited
}

// Check checks the request object.
func (g *GRPCRLSV3) ShouldRateLimit(ctx context.Context, r *pb.RateLimitRequest) (*pb.RateLimitResponse, error) {
	rs := &RLSResponseV3{}

	dlog.Printf(ctx, "shouldRateLimit descriptors: %v\n", r.Descriptors)

	descEntries := make(map[string]string)
	for _, desc := range r.Descriptors {
		for _, entry := range desc.Entries {
			descEntries[entry.Key] = entry.Value
		}
	}

	// Sets overallCode. If x-ambassador-test-allow is present and has value "true", then
	// respond with OK. In any other case, respond with OVER_LIMIT.
	if allowValue := descEntries["x-ambassador-test-allow"]; allowValue == "true" {
		rs.SetOverallCode(pb.RateLimitResponse_OK)
	} else {
		rs.SetOverallCode(pb.RateLimitResponse_OVER_LIMIT)

		// Response headers and body only make sense when the overall code is not OK,
		// so we append them here, if they exist.

		// Append requested headers.
		for _, token := range strings.Split(descEntries["x-ambassador-test-headers-append"], ";") {
			header := strings.Split(strings.TrimSpace(token), "=")
			if len(header) > 1 {
				dlog.Printf(ctx, "appending header %s : %s", header[0], header[1])
				rs.AddHeader(true, header[0], header[1])
			}
		}

		// Set the content-type header, since we're returning json
		rs.AddHeader(true, "content-type", "application/json")
		rs.AddHeader(true, "x-grpc-service-protocol-version", g.ProtocolVersion)

		// Sets results body.
		results := make(map[string]interface{})
		// TODO: Pass back descriptors
		results["descriptors"] = ""
		results["backend"] = g.Backend
		results["status"] = rs.GetOverallCode()
		if rs.GetHTTPHeaderMap() != nil {
			results["headers"] = *rs.GetHTTPHeaderMap()
		}
		body, err := json.MarshalIndent(results, "", "  ")
		if err != nil {
			body = []byte(fmt.Sprintf("Error: %v", err))
		}

		// Sets response body.
		dlog.Printf(ctx, "setting response body: %s", string(body))
		rs.SetBody(string(body))
	}

	return rs.GetResponse(), nil
}

// RLSResponseV3 constructs an rls response object.
type RLSResponseV3 struct {
	headers     []*core.HeaderValueOption
	body        string
	overallCode pb.RateLimitResponse_Code
}

// AddHeader adds a header to the response. When append param is true, Envoy will
// append the value to an existent request header instead of overriding it.
func (r *RLSResponseV3) AddHeader(a bool, k, v string) {
	val := &core.HeaderValueOption{
		Header: &core.HeaderValue{
			Key:   k,
			Value: v,
		},
		Append: &wrappers.BoolValue{Value: a},
	}
	r.headers = append(r.headers, val)
}

// GetHTTPHeaderMap returns HTTP header mapping of the response header-options.
func (r *RLSResponseV3) GetHTTPHeaderMap() *http.Header {
	h := &http.Header{}
	for _, v := range r.headers {
		h.Add(v.Header.Key, v.Header.Value)
	}
	return h
}

// SetBody sets the rls response message body.
func (r *RLSResponseV3) SetBody(s string) {
	r.body = s
}

// SetOverallCode sets the rls response HTTP status code.
func (r *RLSResponseV3) SetOverallCode(code pb.RateLimitResponse_Code) {
	r.overallCode = code
}

// GetOverallCode returns the rls response HTTP status code.
func (r *RLSResponseV3) GetOverallCode() pb.RateLimitResponse_Code {
	return r.overallCode
}

// GetResponse returns the gRPC rls response object.
func (r *RLSResponseV3) GetResponse() *pb.RateLimitResponse {
	rs := &pb.RateLimitResponse{}
	rs.OverallCode = r.overallCode
	rs.RawBody = []byte(r.body)
	for _, h := range r.headers {
		hdr := h.Header
		if hdr != nil {
			rs.ResponseHeadersToAdd = append(rs.ResponseHeadersToAdd,
				&core.HeaderValue{
					Key:   hdr.Key,
					Value: hdr.Value,
				},
			)
		}
	}
	return rs
}
