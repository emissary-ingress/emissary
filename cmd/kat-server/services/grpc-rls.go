package services

import (
	// stdlib
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	// third party
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/wrapperspb"

	// first party (protobuf)
	core "github.com/emissary-ingress/emissary/v3/pkg/api/envoy/api/v2/core"
	pb "github.com/emissary-ingress/emissary/v3/pkg/api/envoy/service/ratelimit/v2"

	// first party
	"github.com/datawire/dlib/dgroup"
	"github.com/datawire/dlib/dhttp"
	"github.com/datawire/dlib/dlog"
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
func (g *GRPCRLS) Start(ctx context.Context) <-chan bool {
	dlog.Printf(ctx, "GRPCRLS: %s listening on %d/%d", g.Backend, g.Port, g.SecurePort)

	grpcHandler := grpc.NewServer()
	dlog.Printf(ctx, "registering v2 service")
	pb.RegisterRateLimitServiceServer(grpcHandler, g)

	cer, err := tls.LoadX509KeyPair(g.Cert, g.Key)
	if err != nil {
		dlog.Error(ctx, err)
		panic(err) // TODO: do something better
	}

	sc := &dhttp.ServerConfig{
		Handler: grpcHandler,
		TLSConfig: &tls.Config{
			Certificates: []tls.Certificate{cer},
		},
	}

	grp := dgroup.NewGroup(ctx, dgroup.GroupConfig{})
	grp.Go("cleartext", func(ctx context.Context) error {
		return sc.ListenAndServe(ctx, fmt.Sprintf(":%v", g.Port))
	})
	grp.Go("tls", func(ctx context.Context) error {
		return sc.ListenAndServeTLS(ctx, fmt.Sprintf(":%v", g.SecurePort), "", "")
	})

	dlog.Print(ctx, "starting gRPC rls service")

	exited := make(chan bool)
	go func() {
		if err := grp.Wait(); err != nil {
			dlog.Error(ctx, err)
			panic(err) // TODO: do something better
		}
		close(exited)
	}()
	return exited
}

// Check checks the request object.
func (g *GRPCRLS) ShouldRateLimit(ctx context.Context, r *pb.RateLimitRequest) (*pb.RateLimitResponse, error) {
	rs := &RLSResponse{}

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
		Append: &wrapperspb.BoolValue{Value: a},
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
