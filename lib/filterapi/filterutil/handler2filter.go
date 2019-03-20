package filterutil

import (
	"context"
	"crypto/tls"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"

	"github.com/pkg/errors"

	"github.com/datawire/apro/lib/filterapi"
)

type responseWriter struct {
	status        int
	header        http.Header
	headerWritten bool
	body          strings.Builder
}

func (rw *responseWriter) Header() http.Header {
	return rw.header
}

func (rw *responseWriter) Write(b []byte) (int, error) {
	if !rw.headerWritten {
		rw.WriteHeader(http.StatusOK)
	}
	return rw.body.Write(b)
}

func (rw *responseWriter) WriteHeader(statusCode int) {
	if rw.headerWritten {
		return
	}
	rw.headerWritten = true
	rw.status = statusCode
}

func (rw *responseWriter) toFilterResponse() filterapi.FilterResponse {
	var ret filterapi.FilterResponse
	if rw.status == http.StatusOK {
		var headers []filterapi.HTTPHeaderModification
		for k, vs := range rw.header {
			for _, v := range vs {
				// TODO(lukeshu): Verify that using ReplaceValue here
				// (as opposed to Appendvalue) matches Envoy's behavior.
				headers = append(headers, &filterapi.HTTPHeaderReplaceValue{
					Key:   k,
					Value: v,
				})
			}
		}
		ret = &filterapi.HTTPRequestModification{
			Header: headers,
		}
	} else {
		ret = &filterapi.HTTPResponse{
			StatusCode: rw.status,
			Header:     rw.header,
			Body:       rw.body.String(),
		}
	}
	return ret
}

func grpcRequestToHTTPServerRequest(g *filterapi.FilterRequest, ctx context.Context) (*http.Request, error) {
	var err error

	body := g.GetRequest().GetHttp().GetBody().String()
	httpVer := g.GetRequest().GetHttp().GetProtocol()
	httpVerMajor, httpVerMinor, ok := http.ParseHTTPVersion(httpVer)
	if !ok {
		err = errors.Errorf("could not parse HTTP version: %q", httpVer)
	}
	h := &http.Request{
		Method: g.GetRequest().GetHttp().GetMethod(),
		//URL: (see below),
		Proto:            httpVer,
		ProtoMajor:       httpVerMajor,
		ProtoMinor:       httpVerMinor,
		Header:           http.Header{}, // see below
		Body:             ioutil.NopCloser(strings.NewReader(body)),
		GetBody:          nil, // ignored for server requests
		ContentLength:    int64(len(body)),
		TransferEncoding: nil,   // supporting this seems like a pain
		Close:            false, // ignored for server requests
		Host:             g.GetRequest().GetHttp().GetHost(),
		Form:             nil,
		PostForm:         nil,
		MultipartForm:    nil,
		Trailer:          nil, // everything is in the Header
		RemoteAddr: fmt.Sprintf("%s:%d",
			g.GetSource().GetAddress().GetSocketAddress().GetAddress(),
			g.GetSource().GetAddress().GetSocketAddress().GetPortValue(),
		),
		RequestURI: g.GetRequest().GetHttp().GetPath(),
		//TLS: (see below),
		Cancel:   nil,
		Response: nil,
	}
	if h.Method == "CONNECT" && !strings.HasPrefix(h.RequestURI, "/") {
		var _err error
		h.URL, _err = url.ParseRequestURI("http://" + h.RequestURI)
		h.URL.Scheme = ""
		if err == nil {
			err = _err
		}
	} else {
		var _err error
		h.URL, _err = url.ParseRequestURI(h.RequestURI)
		if err == nil {
			err = _err
		}
	}
	for k, v := range g.GetRequest().GetHttp().GetHeaders() {
		h.Header.Set(k, v)
	}
	switch scheme := g.GetRequest().GetHttp().GetScheme(); scheme {
	case "http":
		h.TLS = nil
	case "https":
		h.TLS = &tls.ConnectionState{} // just pass a .TLS != nil check
	default:
		if err == nil {
			err = errors.Errorf("unknown scheme: %q", scheme)
		}
	}
	return h.WithContext(ctx), err
}

type httpFilter struct {
	handler http.Handler
}

func (f *httpFilter) Filter(ctx context.Context, gr *filterapi.FilterRequest) (filterapi.FilterResponse, error) {
	hr, _ := grpcRequestToHTTPServerRequest(gr, ctx)
	// TODO(lukeshu): log/handle errors from
	// grpcRequestToHTTPServerRequest.  They should be non-fatal,
	// I think?

	w := &responseWriter{
		status: http.StatusOK,
		header: http.Header{},
	}
	err := func() (err error) {
		defer func() {
			if r := recover(); r != nil {
				err = errors.Errorf("PANIC: %v", r)
			}
		}()
		f.handler.ServeHTTP(w, hr)
		return
	}()
	if err != nil {
		return nil, err
	}
	return w.toFilterResponse(), nil
}

func HandlerToFilter(h http.Handler) filterapi.Filter {
	switch h := h.(type) {
	case filterapi.Filter:
		return h
	default:
		return &httpFilter{handler: h}
	}
}
