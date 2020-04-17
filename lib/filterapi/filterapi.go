// Package filterapi provides convenient type-aliases for all of the
// types involved in writing a Filter.  It abstracts over the need to
// import half a dozen different generated protobuf packages.
//
// Another part of the idea of this package is that at some point we
// might expose it to plugins.
package filterapi

import (
	"context"
	"net/http"

	"github.com/gogo/protobuf/types"
	"google.golang.org/grpc"
	rpc "istio.io/gogo-genproto/googleapis/google/rpc"

	envoyCoreV2 "github.com/datawire/ambassador/pkg/api/envoy/api/v2/core"
	envoyAuthV2 "github.com/datawire/ambassador/pkg/api/envoy/service/auth/v2"
	envoyAuthV2alpha "github.com/datawire/ambassador/pkg/api/envoy/service/auth/v2alpha"
	envoyType "github.com/datawire/ambassador/pkg/api/envoy/type"
)

// RegisterFilterService registers a Filter to the handle the
// "envoy.service.auth.v2alpha.Authorization" and
// "envoy.service.auth.v2.Authorization" services for the gRPC server.
func RegisterFilterService(grpcServer *grpc.Server, filterService Filter) {
	envoyAuthV2alpha.RegisterAuthorizationServer(grpcServer, authorizationService{Filter: filterService})
	envoyAuthV2.RegisterAuthorizationServer(grpcServer, authorizationService{Filter: filterService})
}

type authorizationService struct {
	Filter Filter
}

func (as authorizationService) Check(ctx context.Context, req *envoyAuthV2.CheckRequest) (*envoyAuthV2.CheckResponse, error) {
	filterResponse, err := as.Filter.Filter(ctx, req.GetAttributes())
	if err != nil {
		return nil, err
	}
	return filterResponse.toCheckResponse(), nil
}

type FilterClient interface {
	Filter(ctx context.Context, in *FilterRequest, opts ...grpc.CallOption) (FilterResponse, error)
}

type filterClient struct {
	authorizationClient envoyAuthV2.AuthorizationClient
}

func (fc *filterClient) Filter(ctx context.Context, in *FilterRequest, opts ...grpc.CallOption) (FilterResponse, error) {
	checkResponse, err := fc.authorizationClient.Check(ctx, &envoyAuthV2.CheckRequest{Attributes: in}, opts...)
	if err != nil {
		return nil, err
	}
	if checkResponse.GetStatus().GetCode() == int32(rpc.OK) {
		ret := &HTTPRequestModification{
			Header: nil,
		}
		for _, headerValueOption := range checkResponse.GetOkResponse().GetHeaders() {
			asAppend := true // docs claim this is default https://godoc.org/github.com/datawire/ambassador/pkg/api/envoy/api/v2/core#HeaderValueOption
			if headerValueOption.GetAppend() != nil {
				asAppend = headerValueOption.GetAppend().GetValue()
			}
			if asAppend {
				ret.Header = append(ret.Header, &HTTPHeaderAppendValue{
					Key:   headerValueOption.GetHeader().GetKey(),
					Value: headerValueOption.GetHeader().GetValue(),
				})
			} else {
				ret.Header = append(ret.Header, &HTTPHeaderReplaceValue{
					Key:   headerValueOption.GetHeader().GetKey(),
					Value: headerValueOption.GetHeader().GetValue(),
				})
			}
		}
		return ret, nil
	} else {
		ret := &HTTPResponse{
			StatusCode: int(checkResponse.GetDeniedResponse().GetStatus().GetCode()),
			Header:     http.Header{},
			Body:       checkResponse.GetDeniedResponse().GetBody(),
		}
		for _, headerValueOption := range checkResponse.GetDeniedResponse().GetHeaders() {
			ret.Header.Add(headerValueOption.GetHeader().GetKey(), headerValueOption.GetHeader().GetValue())
		}
		return ret, nil
	}
}

func NewFilterClient(cc *grpc.ClientConn) FilterClient {
	return &filterClient{authorizationClient: envoyAuthV2.NewAuthorizationClient(cc)}
}

// A Filter is something that can modify or intercept an incoming HTTP
// request.
type Filter interface {
	Filter(context.Context, *FilterRequest) (FilterResponse, error)
}

// FilterRequest represents a request to a filter.
//
//     type FilterRequest struct {
//
//         // The source of a network activity, such as starting a TCP connection.
//         // In a multi hop network activity, the source represents the sender of the
//         // last hop.
//         Source *envoyAuthV2.AttributeContext_Peer
//
//         // The destination of a network activity, such as accepting a TCP connection.
//         // In a multi hop network activity, the destination represents the receiver of
//         // the last hop.
//         Destination *envoyAuthV2.AttributeContext_Peer
//
//         // Represents a network request, such as an HTTP request.
//         Request *envoyAuthV2.AttributeContext_Request
//
//         // This is analogous to http_request.headers, however these contents will not be sent to the
//         // upstream server. Context_extensions provide an extension mechanism for sending additional
//         // information to the auth server without modifying the proto definition. It maps to the
//         // internal opaque context in the filter chain.
//         ContextExtensions map[string]string
//     }
//
// Whether or not request.GetRequest().GetHttp().GetBody().String() is
// set depends on whether `allow_request_body` is true or false in the
// Ambassador AuthService YAML.
//
// TODO(lukeshu): Consider defining this as a more convenient struct,
// and translating envoyAuthV2.AttributeContext to it, instead of just
// type-aliasing it.
type FilterRequest = envoyAuthV2.AttributeContext

// FilterResponse is is a response that a Filter can return; is it is
// implemented by both "HTTPRequestModification" and by
// "HTTPResponse".
type FilterResponse interface {
	toCheckResponse() *envoyAuthV2.CheckResponse
}

// HTTPRequestModification is a FilterResponse that modifies the HTTP
// request before passing it along to the next Filter or the backend
// service.
type HTTPRequestModification struct {
	Header []HTTPHeaderModification
}

// This is a compile-time check that HTTPRequestModification does indeed implement
// the FilterResponse interface.
var _ FilterResponse = &HTTPRequestModification{}

type HTTPHeaderModification interface {
	toHeaderValueOption() *envoyCoreV2.HeaderValueOption
}

type HTTPHeaderAppendValue struct {
	Key   string
	Value string
}

var _ HTTPHeaderModification = &HTTPHeaderAppendValue{} // Another compile-time type check.

// use a pointer-receiver so that code doing a type-switch doesn't
// need to handle both the pointer and non-pointer cases; force
// everything to be a pointer.
func (h *HTTPHeaderAppendValue) toHeaderValueOption() *envoyCoreV2.HeaderValueOption {
	return &envoyCoreV2.HeaderValueOption{
		Header: &envoyCoreV2.HeaderValue{
			Key:   h.Key,
			Value: h.Value,
		},
		Append: &types.BoolValue{Value: true},
	}
}

type HTTPHeaderReplaceValue struct {
	Key   string
	Value string
}

var _ HTTPHeaderModification = &HTTPHeaderReplaceValue{}

// use a pointer-receiver so that code doing a type-switch doesn't
// need to handle both the pointer and non-pointer cases; force
// everything to be a pointer.
func (h *HTTPHeaderReplaceValue) toHeaderValueOption() *envoyCoreV2.HeaderValueOption {
	return &envoyCoreV2.HeaderValueOption{
		Header: &envoyCoreV2.HeaderValue{
			Key:   h.Key,
			Value: h.Value,
		},
		Append: &types.BoolValue{Value: false},
	}
}

// use a pointer-receiver so that code doing a type-switch doesn't
// need to handle both the pointer and non-pointer cases; force
// everything to be a pointer.
func (r *HTTPRequestModification) toCheckResponse() *envoyAuthV2.CheckResponse {
	headers := make([]*envoyCoreV2.HeaderValueOption, len(r.Header))
	for i := range r.Header {
		headers[i] = r.Header[i].toHeaderValueOption()
	}
	return &envoyAuthV2.CheckResponse{
		Status: &rpc.Status{Code: int32(rpc.OK)},
		HttpResponse: &envoyAuthV2.CheckResponse_OkResponse{
			OkResponse: &envoyAuthV2.OkHttpResponse{
				Headers: headers,
			},
		},
	}
}

// HTTPResponse is a FilterResponse that provides an HTTP response to
// send back to the client; the HTTP request is not passed along to
// the backend service or any more Filters.
type HTTPResponse struct {
	StatusCode int
	Header     http.Header
	Body       string
}

var _ FilterResponse = &HTTPResponse{}

// use a pointer-receiver so that code doing a type-switch doesn't
// need to handle both the pointer and non-pointer cases; force
// everything to be a pointer.
func (r *HTTPResponse) toCheckResponse() *envoyAuthV2.CheckResponse {
	var headers []*envoyCoreV2.HeaderValueOption
	for k, vs := range r.Header {
		for _, v := range vs {
			headers = append(headers, &envoyCoreV2.HeaderValueOption{
				Header: &envoyCoreV2.HeaderValue{
					Key:   k,
					Value: v,
				},
				// TODO(lukeshu): I'm pretty sure Append is ignored here; verify if that's true
			})
		}
	}
	return &envoyAuthV2.CheckResponse{
		Status: &rpc.Status{Code: int32(rpc.UNAUTHENTICATED)},
		HttpResponse: &envoyAuthV2.CheckResponse_DeniedResponse{
			DeniedResponse: &envoyAuthV2.DeniedHttpResponse{
				Status: &envoyType.HttpStatus{
					Code: envoyType.StatusCode(r.StatusCode),
				},
				Headers: headers,
				Body:    r.Body,
			},
		},
	}
}
