package services

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"

	"strings"

	core "github.com/datawire/ambassador/pkg/api/envoy/api/v2/core"
	pb "github.com/datawire/ambassador/pkg/api/envoy/service/ratelimit/v2"
	pb_legacy "github.com/datawire/ambassador/pkg/api/pb/lyft/ratelimit"

	"github.com/golang/protobuf/ptypes/wrappers"
	"google.golang.org/grpc"
)

// GRPCRLS server object (all fields are required).
type GRPCRLS struct {
	Port            int16
	Backend         string
	SecurePort      int16
	SecureBackend   string
	Cert            string
	Key             string
	ProtocolVersion string
}

// Start initializes the HTTP server.
func (g *GRPCRLS) Start() <-chan bool {
	log.Printf("GRPCRLS: %s listening on %d/%d", g.Backend, g.Port, g.SecurePort)

	exited := make(chan bool)
	proto := "tcp"

	go func() {
		port := fmt.Sprintf(":%v", g.Port)

		ln, err := net.Listen(proto, port)
		if err != nil {
			log.Fatal()
		}

		s := grpc.NewServer()
		if g.ProtocolVersion != "v2" {
			log.Printf("registering v2alpha service")
			pb_legacy.RegisterRateLimitServiceServer(s, g)
		} else {
			log.Printf("registering v2 service")
			pb.RegisterRateLimitServiceServer(s, g)
		}
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
		if g.ProtocolVersion != "v2" {
			log.Printf("registering v2alpha service")
			pb_legacy.RegisterRateLimitServiceServer(s, g)
		} else {
			log.Printf("registering v2 service")
			pb.RegisterRateLimitServiceServer(s, g)
		}
		s.Serve(ln)

		defer ln.Close()
		close(exited)
	}()

	log.Print("starting gRPC rls service")
	return exited
}

// Check checks the request object.
func (g *GRPCRLS) ShouldRateLimit(ctx context.Context, r *pb.RateLimitRequest) (*pb.RateLimitResponse, error) {
	rs := &RLSResponse{}

	log.Printf("shouldRateLimit descriptors: %v\n", r.Descriptors)

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
				log.Printf("appending header %s : %s", header[0], header[1])
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
		log.Printf("setting response body: %s", string(body))
		rs.SetBody(string(body))
	}

	return rs.GetResponse(), nil
}

// RLSResponse constructs an rls response object.
type RLSResponse struct {
	headers     []*core.HeaderValueOption
	body        string
	overallCode pb.RateLimitResponse_Code
}

// AddHeader adds a header to the response. When append param is true, Envoy will
// append the value to an existent request header instead of overriding it.
func (r *RLSResponse) AddHeader(a bool, k, v string) {
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
func (r *RLSResponse) GetHTTPHeaderMap() *http.Header {
	h := &http.Header{}
	for _, v := range r.headers {
		h.Add(v.Header.Key, v.Header.Value)
	}
	return h
}

// SetBody sets the rls response message body.
func (r *RLSResponse) SetBody(s string) {
	r.body = s
}

// SetOverallCode sets the rls response HTTP status code.
func (r *RLSResponse) SetOverallCode(code pb.RateLimitResponse_Code) {
	r.overallCode = code
}

// GetOverallCode returns the rls response HTTP status code.
func (r *RLSResponse) GetOverallCode() pb.RateLimitResponse_Code {
	return r.overallCode
}

// GetResponse returns the gRPC rls response object.
func (r *RLSResponse) GetResponse() *pb.RateLimitResponse {
	rs := &pb.RateLimitResponse{}
	rs.OverallCode = r.overallCode
	rs.RawBody = []byte(r.body)
	for _, h := range r.headers {
		hdr := h.Header
		if hdr != nil {
			rs.Headers = append(rs.Headers,
				&core.HeaderValue{
					Key:   hdr.Key,
					Value: hdr.Value,
				},
			)
		}
	}
	return rs
}
