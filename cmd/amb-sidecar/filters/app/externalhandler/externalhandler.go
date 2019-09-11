package externalhandler

import (
	"context"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/pkg/errors"
	"google.golang.org/grpc"

	crd "github.com/datawire/apro/apis/getambassador.io/v1beta2"
	"github.com/datawire/apro/cmd/amb-sidecar/filters/app/httpclient"
	"github.com/datawire/apro/cmd/amb-sidecar/filters/app/middleware"
	"github.com/datawire/apro/lib/filterapi"
)

func grpcRequestToHTTPClientRequest(g *filterapi.FilterRequest, serviceURL string, allowedHeaders []string, allowRequestBody bool, ctx context.Context) (*http.Request, error) {
	var err error

	var body string
	if allowRequestBody {
		body = g.GetRequest().GetHttp().GetBody()
	}
	h := &http.Request{
		Method:           g.GetRequest().GetHttp().GetMethod(),
		URL:              nil,           // see below
		Proto:            "",            // ignored for client requests
		ProtoMajor:       0,             // ignored for client requests
		ProtoMinor:       0,             // ignored for client requests
		Header:           http.Header{}, // see below
		Body:             ioutil.NopCloser(strings.NewReader(body)),
		GetBody:          func() (io.ReadCloser, error) { return ioutil.NopCloser(strings.NewReader(body)), nil },
		ContentLength:    int64(len(body)),
		TransferEncoding: nil, // supporting this seems like a pain
		Close:            false,
		Host:             g.GetRequest().GetHttp().GetHost(),
		Form:             nil, // ignored for client requests
		PostForm:         nil, // ignored for client requests
		MultipartForm:    nil, // ignored for client requests
		Trailer:          nil, // everything is in the Header
		RemoteAddr:       "",  // ignored for client requests
		RequestURI:       "",  // ignored for client requests
		TLS:              nil, // ignored for client requests
		Cancel:           nil,
		Response:         nil, // set by net/http
	}
	// .URL
	if !strings.HasPrefix(g.GetRequest().GetHttp().GetPath(), "/") {
		return nil, errors.New("I have no idea what to do when the path doesn't start with '/'")
	}
	h.URL, err = url.Parse(serviceURL + g.GetRequest().GetHttp().GetPath())
	if err != nil {
		return nil, errors.Wrap(err, "failed to construct request URL")
	}
	// .Header
	allowedHeadersMap := make(map[string]struct{}, len(allowedHeaders))
	for _, k := range allowedHeaders {
		allowedHeadersMap[k] = struct{}{}
	}
	for k, v := range g.GetRequest().GetHttp().GetHeaders() {
		if _, ok := allowedHeadersMap[http.CanonicalHeaderKey(k)]; ok {
			h.Header.Set(k, v)
		}
	}

	return h.WithContext(ctx), nil
}

func httpResponseToGRPCResponse(h *http.Response, allowedHeaders []string) (filterapi.FilterResponse, error) {
	if h.StatusCode == http.StatusOK {
		allowedHeadersMap := make(map[string]struct{}, len(allowedHeaders))
		for _, k := range allowedHeaders {
			allowedHeadersMap[k] = struct{}{}
		}
		var headers []filterapi.HTTPHeaderModification
		for k, vs := range h.Header {
			if _, ok := allowedHeadersMap[http.CanonicalHeaderKey(k)]; !ok {
				continue
			}
			for _, v := range vs {
				// TODO(lukeshu): Verify that using ReplaceValue here
				// (as opposed to Appendvalue) matches Envoy's behavior.
				headers = append(headers, &filterapi.HTTPHeaderReplaceValue{
					Key:   k,
					Value: v,
				})
			}
		}
		return &filterapi.HTTPRequestModification{
			Header: headers,
		}, nil
	} else {
		body, err := ioutil.ReadAll(h.Body)
		if err != nil {
			return nil, err
		}
		return &filterapi.HTTPResponse{
			StatusCode: h.StatusCode,
			Header:     h.Header,
			Body:       string(body),
		}, nil
	}
}

type ExternalFilter struct {
	Spec crd.FilterExternal
}

func (f *ExternalFilter) Filter(ctx context.Context, r *filterapi.FilterRequest) (filterapi.FilterResponse, error) {
	logger := middleware.GetLogger(ctx)
	ctx, ctxCancel := context.WithTimeout(ctx, f.Spec.Timeout)
	defer ctxCancel()

	serviceURL, err := url.Parse("random://" + f.Spec.AuthService)
	if err != nil {
		return nil, err
	}
	serviceHost := serviceURL.Hostname()
	servicePort := serviceURL.Port()
	if servicePort == "" {
		if f.Spec.TLS {
			servicePort = "443"
		} else {
			servicePort = "80"
		}
	} else if _, err := strconv.Atoi(servicePort); err != nil {
		return nil, errors.Wrap(err, "bad port number")
	}
	serviceAuthority := net.JoinHostPort(serviceHost, servicePort)

	switch f.Spec.Proto {
	case "grpc":
		var dialOptions []grpc.DialOption
		if !f.Spec.TLS {
			dialOptions = append(dialOptions, grpc.WithInsecure())
		}
		grpcClientConn, err := grpc.DialContext(ctx, "dns:///"+serviceAuthority, dialOptions...)
		if err != nil {
			return nil, err
		}
		defer grpcClientConn.Close()
		if !f.Spec.AllowRequestBody {
			_body := r.Request.Http.Body
			r.Request.Http.Body = ""
			defer func() { r.Request.Http.Body = _body }()
		}
		return filterapi.NewFilterClient(grpcClientConn).Filter(ctx, r)
	case "http":
		var serviceURL string
		if f.Spec.TLS {
			serviceURL = "https://" + serviceAuthority
		} else {
			serviceURL = "http://" + serviceAuthority
		}
		serviceURL += f.Spec.PathPrefix

		httpRequest, err := grpcRequestToHTTPClientRequest(r, serviceURL, f.Spec.AllowedRequestHeaders, f.Spec.AllowRequestBody, ctx)
		if err != nil {
			return nil, err
		}

		client := httpclient.NewHTTPClient(logger, 0, false)
		client.CheckRedirect = func(_ *http.Request, _ []*http.Request) error {
			return http.ErrUseLastResponse
		}

		httpResponse, err := client.Do(httpRequest)
		if err != nil {
			return nil, err
		}
		defer httpResponse.Body.Close()

		return httpResponseToGRPCResponse(httpResponse, f.Spec.AllowedAuthorizationHeaders)
	default:
		panic("should not happen")
	}
}
