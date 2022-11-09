package services

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/datawire/dlib/dgroup"
	"github.com/datawire/dlib/dhttp"
	"github.com/datawire/dlib/dlog"
)

// HTTP server object (all fields are required).
type HealthCheckServer struct {
	Port                int16
	Backend             string
	SecurePort          int16
	SecureBackend       string
	Cert                string
	Key                 string
	TLSVersion          string
	Healthy             bool
	HealthyStatusCode   int
	UnhealthyStatusCode int
}

// Start initializes the Health Check HTTP server.
func (h *HealthCheckServer) Start(ctx context.Context) <-chan bool {
	dlog.Printf(ctx, "HTTP: %s listening on %d/%d", h.Backend, h.Port, h.SecurePort)

	h.Healthy = true
	mux := http.NewServeMux()
	mux.HandleFunc("/", h.handler)

	sc := &dhttp.ServerConfig{
		Handler: mux,
	}

	g := dgroup.NewGroup(ctx, dgroup.GroupConfig{})
	g.Go("cleartext", func(ctx context.Context) error {
		return sc.ListenAndServe(ctx, fmt.Sprintf(":%v", h.Port))
	})
	g.Go("tls", func(ctx context.Context) error {
		return sc.ListenAndServeTLS(ctx, fmt.Sprintf(":%v", h.SecurePort), h.Cert, h.Key)
	})

	exited := make(chan bool)
	go func() {
		if err := g.Wait(); err != nil {
			dlog.Error(ctx, err)
			panic(err) // TODO: do something better
		}
		close(exited)
	}()
	return exited
}

func (h *HealthCheckServer) handler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	// Assume we're the clear side of the world.
	backend := h.Backend
	conntype := "CLR"

	var request = make(map[string]interface{})
	var url = make(map[string]interface{})
	request["url"] = url
	url["fragment"] = r.URL.Fragment
	url["host"] = r.URL.Host
	url["opaque"] = r.URL.Opaque
	url["path"] = r.URL.Path
	url["query"] = r.URL.Query()
	url["rawQuery"] = r.URL.RawQuery
	url["scheme"] = r.URL.Scheme
	if r.URL.User != nil {
		url["username"] = r.URL.User.Username()
		pw, ok := r.URL.User.Password()
		if ok {
			url["password"] = pw
		}
	}
	request["method"] = r.Method
	request["headers"] = lower(r.Header)
	request["host"] = r.Host

	var tlsrequest = make(map[string]interface{})
	request["tls"] = tlsrequest

	tlsrequest["enabled"] = r.TLS != nil

	if r.TLS != nil {
		// We're the secure side of the world, I guess.
		backend = h.SecureBackend
		conntype = "TLS"

		tlsrequest["negotiated-protocol"] = r.TLS.NegotiatedProtocol
		tlsrequest["server-name"] = r.TLS.ServerName
		tlsrequest["negotiated-protocol-version"] = getTLSVersion(r.TLS)
	}

	// Set date response header.
	w.Header().Set("Date", time.Now().Format(time.RFC1123))

	statusCode := h.HealthyStatusCode
	if !h.Healthy {
		statusCode = h.UnhealthyStatusCode
	}

	fmt.Println(r.URL.Path)
	// A request to this path will make the health check server respond with
	// only the UnhealthyStatusCode to all subsequent requests
	if r.URL.Path == "/makeUnhealthy/" {
		h.Healthy = false
	}

	w.WriteHeader(statusCode)

	// Write out all request/response information
	var response = make(map[string]interface{})
	response["headers"] = lower(w.Header())

	var body = make(map[string]interface{})
	body["backend"] = backend
	body["request"] = request
	body["response"] = response

	b, err := json.MarshalIndent(body, "", "  ")
	if err != nil {
		b = []byte(fmt.Sprintf("Error: %v", err))
	}

	dlog.Printf(ctx, "%s (%s): \"%s %s\" -> HTTP %v", r.Method, r.URL.Path, backend, conntype, statusCode)
	_, _ = w.Write(b)
}
