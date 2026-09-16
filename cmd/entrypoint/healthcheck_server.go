package entrypoint

import (
	"context"
	"net"
	"net/http"
	"net/http/httputil"
	"net/http/pprof"
	"net/url"

	_ "k8s.io/client-go/plugin/pkg/client/auth"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"

	"github.com/datawire/dlib/dhttp"
	"github.com/emissary-ingress/emissary/v3/pkg/acp"
	"github.com/emissary-ingress/emissary/v3/pkg/debug"
)

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

	// diagd's internal API -- most importantly /_internal/v0/watt, which is how a
	// new snapshot gets submitted -- is flatly not available here. This port is
	// reachable from off-pod, and `kubectl port-forward` makes off-pod traffic look
	// like it came from 127.0.0.1, so there is no way to tell local traffic from
	// remote traffic here. Anything that needs the internal API has to talk to
	// diagd over its Unix-domain socket instead.
	//
	// Note that this has to be registered before the "/" catchall below, and that
	// ServeMux normalizes paths (collapsing "..", etc.) before matching, so this
	// can't be walked around with a cleverly-encoded path.
	sm.HandleFunc("/_internal/", func(w http.ResponseWriter, r *http.Request) {
		http.NotFound(w, r)
	})

	// For everything else, use a ReverseProxy to forward it to diagd.
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

	// Finally, use the reverseProxy to handle anything coming in on
	// the magic catchall path.
	sm.HandleFunc("/", reverseProxy.ServeHTTP)

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
