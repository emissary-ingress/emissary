package entrypoint

import (
	"context"
	"net"
	"net/http"
	"net/http/httputil"
	"net/http/pprof"
	"net/url"
	"path"
	"strings"

	_ "k8s.io/client-go/plugin/pkg/client/auth"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"

	"github.com/datawire/dlib/dhttp"
	"github.com/emissary-ingress/emissary/v3/pkg/acp"
	"github.com/emissary-ingress/emissary/v3/pkg/debug"
)

// diagdPathAllowed reports whether p is a path we're willing to hand to diagd on
// the health check port. p must be the *decoded* path: ServeMux matches patterns
// against the escaped path, but gunicorn decodes %2f before Werkzeug routes on
// the result, so the two disagree on where a request is going and both have to
// agree it's allowed.
func diagdPathAllowed(p string) bool {
	// Clean resolves the ".." that an encoded separator can smuggle in, and drops
	// any trailing slash.
	p = path.Clean(p)

	return p == "/metrics" ||
		p == "/ambassador/v0/favicon.ico" ||
		p == "/ambassador/v0/diag" ||
		strings.HasPrefix(p, "/ambassador/v0/diag/")
}

func handleCheckAlive(w http.ResponseWriter, r *http.Request, ambwatch *acp.AmbassadorWatcher) {
	// The liveness check needs to explicitly try to talk to Envoy...
	ambwatch.FetchEnvoyReady(r.Context())

	// ...then check if the watcher says we're alive.
	ok := ambwatch.IsAlive()

	if ok {
		_, _ = w.Write([]byte("Ambassador is alive and well\n"))
	} else {
		http.Error(w, "Ambassador is not alive\n", http.StatusServiceUnavailable)
	}
}

func handleCheckReady(w http.ResponseWriter, r *http.Request, ambwatch *acp.AmbassadorWatcher) {
	// The readiness check needs to explicitly try to talk to Envoy, too. Why?
	// Because if you have a pod configured with only the readiness check but
	// not the liveness check, and we don't try to talk to Envoy here, then we
	// will never ever attempt to talk to Envoy at all, Envoy will never be
	// declared alive, and we'll never consider Ambassador ready.
	ambwatch.FetchEnvoyReady(r.Context())

	ok := ambwatch.IsReady()

	if ok {
		_, _ = w.Write([]byte("Ambassador is ready and waiting\n"))
	} else {
		http.Error(w, "Ambassador is not ready\n", http.StatusServiceUnavailable)
	}
}

// healthCheckMux builds the handler for the health check port (8877 by default).
// This port is reachable from off-pod, so be very careful about what gets
// registered here.
func healthCheckMux(ctx context.Context, ambwatch *acp.AmbassadorWatcher) *http.ServeMux {
	dbg := debug.FromContext(ctx)

	// We need to do some HTTP stuff by hand to catch the readiness and liveness
	// checks here, but forward everything else to diagd.
	sm := http.NewServeMux()

	// Handle the liveness check and the readiness check directly, by handing them
	// off to our functions.

	livenessTimer := dbg.Timer("check_alive")
	sm.HandleFunc("/ambassador/v0/check_alive",
		livenessTimer.TimedHandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			handleCheckAlive(w, r, ambwatch)
		}))

	readinessTimer := dbg.Timer("check_ready")
	sm.HandleFunc("/ambassador/v0/check_ready",
		readinessTimer.TimedHandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			handleCheckReady(w, r, ambwatch)
		}))

	// Serve any debug info from the golang codebase.
	sm.Handle("/debug", dbg)

	// Serve pprof endpoints to aid in live debugging.
	sm.HandleFunc("/debug/pprof/", pprof.Index)
	sm.HandleFunc("/debug/pprof/profile", pprof.Profile)
	sm.HandleFunc("/debug/pprof/trace", pprof.Trace)
	sm.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
	sm.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)

	// Proxy to diagd for the handful of paths diagd serves that belong on this
	// port.
	//
	// diagd listens on a Unix-domain socket, so diagdOrigin is just a placeholder
	// host: DiagdTransport dials the socket no matter what's in the URL.
	diagdOrigin, _ := url.Parse(DiagdURLOrigin)

	// This reverseProxy is dirt simple: use a director function to
	// swap the scheme and host of our request for the ones from the
	// diagdOrigin. Leave everything else (notably including the path)
	// alone.
	reverseProxy := &httputil.ReverseProxy{
		Transport: DiagdTransport(),
		Director: func(req *http.Request) {
			req.URL.Scheme = diagdOrigin.Scheme
			req.URL.Host = diagdOrigin.Host

			// Whatever the client had to say about X-Ambassador-Diag-IP, they don't
			// get a vote: this is a header that we generate, and honoring a
			// client-supplied value would let anyone who can reach this port claim
			// to be local. Drop it before we consider setting it ourselves.
			req.Header.Del("X-Ambassador-Diag-IP")

			// If this request is coming from localhost, tell diagd about that.
			if acp.HostPortIsLocal(req.RemoteAddr) {
				req.Header.Set("X-Ambassador-Diag-IP", "127.0.0.1")
			}
		},
	}

	// This is an allowlist, and it needs to stay one. The rest of what diagd
	// serves is its internal API -- /_internal/v0/{ping,features,watt,fs}, which
	// includes snapshot submission -- and this port is reachable from off-pod.
	// `kubectl port-forward` makes off-pod traffic look like it came from
	// 127.0.0.1, so there is no way to tell local from remote here; the only safe
	// answer is that the internal API is not routable from this mux at all. It is
	// reachable solely over diagd's Unix-domain socket.
	//
	// Listing what's allowed rather than excluding what isn't also sidesteps
	// encoded-separator tricks. Go matches patterns against the *escaped* path, so
	// "/_internal%2fv0%2fwatt" is a single segment that matches nothing here --
	// but gunicorn decodes %2f before Werkzeug routes, so a catchall would have
	// handed exactly that to the internal handler.
	//
	// A new diagd route does not appear here by default, which is the point: it
	// fails closed, and adding it is a deliberate act.
	proxyAllowed := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !diagdPathAllowed(r.URL.Path) {
			http.NotFound(w, r)
			return
		}
		reverseProxy.ServeHTTP(w, r)
	})

	sm.Handle("/metrics", proxyAllowed)
	sm.Handle("/ambassador/v0/favicon.ico", proxyAllowed)
	sm.Handle("/ambassador/v0/diag/", proxyAllowed)

	return sm
}

func healthCheckHandler(ctx context.Context, ambwatch *acp.AmbassadorWatcher) error {
	sm := healthCheckMux(ctx, ambwatch)

	// Set up listener.
	// The default value for network is ANY.
	// It means in case of any wildcard address, the listener will try to listen on both IPv4 and IPv6
	// If you want to specify AF explicitly,
	// you can set the AMBASSADOR_HEALTHCHECK_IP_FAMILY environment variable to IPV4_ONLY or IPV6_ONLY respectively
	addr := net.JoinHostPort(getHealthCheckHost(), getHealthCheckPort())
	network := getHealthCheckIPNetworkFamily()
	listener, err := net.Listen(network, addr)
	if err != nil {
		return err
	}

	s := &dhttp.ServerConfig{
		Handler: sm,
	}

	return s.Serve(ctx, listener)
}
