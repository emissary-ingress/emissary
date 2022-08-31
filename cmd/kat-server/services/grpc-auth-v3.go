package services

import (
	// stdlib
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	// third party
	"google.golang.org/genproto/googleapis/rpc/code"
	"google.golang.org/genproto/googleapis/rpc/status"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/wrapperspb"

	// first party (protobuf)
	apiv3_core "github.com/emissary-ingress/emissary/v3/pkg/api/envoy/config/core/v3"
	apiv3_svc_auth "github.com/emissary-ingress/emissary/v3/pkg/api/envoy/service/auth/v3"
	apiv3_type "github.com/emissary-ingress/emissary/v3/pkg/api/envoy/type/v3"

	// first party
	"github.com/datawire/dlib/dgroup"
	"github.com/datawire/dlib/dhttp"
	"github.com/datawire/dlib/dlog"
)

// GRPCAuthV3 server object (all fields are required).
type GRPCAuthV3 struct {
	Port            int16
	Backend         string
	SecurePort      int16
	SecureBackend   string
	Cert            string
	Key             string
	ProtocolVersion string
}

// Start initializes the HTTP server.
func (g *GRPCAuthV3) Start(ctx context.Context) <-chan bool {
	dlog.Printf(ctx, "GRPCAuthV3: %s listening on %d/%d", g.Backend, g.Port, g.SecurePort)

	grpcHandler := grpc.NewServer()
	dlog.Printf(ctx, "registering v3 service")
	apiv3_svc_auth.RegisterAuthorizationServer(grpcHandler, g)

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

	dlog.Print(ctx, "starting gRPC authorization service")

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
func (g *GRPCAuthV3) Check(ctx context.Context, r *apiv3_svc_auth.CheckRequest) (*apiv3_svc_auth.CheckResponse, error) {
	rs := &ResponseV3{}

	rheader := r.GetAttributes().GetRequest().GetHttp().GetHeaders()
	rbody := r.GetAttributes().GetRequest().GetHttp().GetBody()
	if len(rbody) > 0 {
		rheader["body"] = rbody
	}

	rContextExtensions := r.GetAttributes().GetContextExtensions()
	if rContextExtensions != nil {
		val, err := json.Marshal(rContextExtensions)
		if err != nil {
			val = []byte(fmt.Sprintf("Error: %v", err))
		}

		rs.AddHeader(false, "kat-resp-extauth-context-extensions", string(val))
	}

	// Sets requested HTTP status.
	rs.SetStatus(ctx, rheader["kat-req-extauth-requested-status"])

	rs.AddHeader(false, "kat-resp-extauth-protocol-version", g.ProtocolVersion)

	// Sets requested headers.
	// Don't bother if we'll be returning a apiv3_svc_auth.CheckResponse_OkResponse; it'd be a no-op in that case.
	if rs.status != http.StatusOK && rs.status != 0 {
		for _, key := range strings.Split(strings.ToLower(rheader["kat-req-extauth-requested-header"]), ",") {
			if val := rheader[key]; val != "" {
				rs.AddHeader(false, key, val)
			}
		}
	}

	// Append requested headers.
	for _, token := range strings.Split(rheader["kat-req-extauth-append"], ";") {
		header := strings.Split(strings.TrimSpace(token), "=")
		if len(header) > 1 {
			dlog.Printf(ctx, "appending header %s : %s", header[0], header[1])
			rs.AddHeader(true, header[0], header[1])
		}
	}

	// Sets requested Cookies.
	for _, v := range strings.Split(rheader["kat-req-extauth-requested-cookie"], ",") {
		val := strings.Trim(v, " ")
		rs.AddHeader(false, "Set-Cookie", fmt.Sprintf("%s=%s", val, val))
	}

	// Sets requested location.
	if loc := rheader["kat-req-extauth-requested-location"]; loc != "" {
		rs.AddHeader(false, "Location", loc)
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
	dlog.Printf(ctx, "setting response body: %s", string(body))
	rs.SetBody(string(body))

	return rs.GetResponse(), nil
}

// ResponseV3 constructs an authorization response object.
type ResponseV3 struct {
	headers []*apiv3_core.HeaderValueOption
	body    string
	status  uint32
}

// AddHeader adds a header to the response. When append param is true, Envoy will
// append the value to an existent request header instead of overriding it.
func (r *ResponseV3) AddHeader(a bool, k, v string) {
	val := &apiv3_core.HeaderValueOption{
		Header: &apiv3_core.HeaderValue{
			Key:   k,
			Value: v,
		},
		Append: &wrapperspb.BoolValue{Value: a},
	}
	r.headers = append(r.headers, val)
}

// GetHTTPHeaderMap returns HTTP header mapping of the response header-options.
func (r *ResponseV3) GetHTTPHeaderMap() *http.Header {
	h := &http.Header{}
	for _, v := range r.headers {
		h.Add(v.Header.Key, v.Header.Value)
	}
	return h
}

// SetBody sets the authorization response message body.
func (r *ResponseV3) SetBody(s string) {
	r.body = s
}

// SetStatus sets the authorization response HTTP status code.
func (r *ResponseV3) SetStatus(ctx context.Context, s string) {
	if len(s) == 0 {
		s = "200"
	}
	if val, err := strconv.Atoi(s); err == nil {
		r.status = uint32(val)
		r.AddHeader(false, "status", s)
		dlog.Printf(ctx, "setting HTTP status %v", r.status)
	} else {
		r.status = uint32(500)
		r.AddHeader(false, "status", "500")
		dlog.Printf(ctx, "error setting HTTP status. Cannot parse string %s: %v.", s, err)
	}
}

// GetStatus returns the authorization response HTTP status code.
func (r *ResponseV3) GetStatus() uint32 {
	return r.status
}

// GetResponse returns the gRPC authorization response object.
func (r *ResponseV3) GetResponse() *apiv3_svc_auth.CheckResponse {
	rs := &apiv3_svc_auth.CheckResponse{}
	switch {
	// Ok respose.
	case r.status == http.StatusOK || r.status == 0:
		rs.Status = &status.Status{Code: int32(code.Code_OK)}
		rs.HttpResponse = &apiv3_svc_auth.CheckResponse_OkResponse{
			OkResponse: &apiv3_svc_auth.OkHttpResponse{
				Headers: r.headers,
			},
		}

	// Denied response.
	default:
		rs.Status = &status.Status{Code: int32(code.Code_UNAUTHENTICATED)}
		rs.HttpResponse = &apiv3_svc_auth.CheckResponse_DeniedResponse{
			DeniedResponse: &apiv3_svc_auth.DeniedHttpResponse{
				Status: &apiv3_type.HttpStatus{
					Code: apiv3_type.StatusCode(r.status),
				},
				Headers: r.headers,
				Body:    r.body,
			},
		}
	}

	return rs
}
