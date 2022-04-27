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

func healthCheckHandler(ctx context.Context, ambwatch *acp.AmbassadorWatcher) error {
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

	// For everything else, use a ReverseProxy to forward it to diagd.
	//
	// diagdOrigin is where diagd is listening.
	diagdOrigin, _ := url.Parse("http://127.0.0.1:8004/")

	// This reverseProxy is dirt simple: use a director function to
	// swap the scheme and host of our request for the ones from the
	// diagdOrigin. Leave everything else (notably including the path)
	// alone.
	reverseProxy := &httputil.ReverseProxy{
		Director: func(req *http.Request) {
			req.URL.Scheme = diagdOrigin.Scheme
			req.URL.Host = diagdOrigin.Host

			// If this request is coming from localhost, tell diagd about that.
			if acp.HostPortIsLocal(req.RemoteAddr) {
				req.Header.Set("X-Ambassador-Diag-IP", "127.0.0.1")
			}
		},
	}

	// Finally, use the reverseProxy to handle anything coming in on
	// the magic catchall path.
	sm.HandleFunc("/", reverseProxy.ServeHTTP)

	// Create a listener by hand, so that we can listen on TCP v4. If we don't
	// explicitly say "tcp4" here, we seem to listen _only_ on v6, and Bad Things
	// Happen.
	//
	// XXX Why, exactly, is this? That's a lovely question -- we _should_ be OK
	// here on a proper dualstack system, but apparently we don't have a proper
	// dualstack system? It's quite bizarre, but Kubernetes won't become ready
	// without this.
	//
	// XXX In fact, should we set up another Listener for v6??
	listener, err := net.Listen("tcp4", ":8877")
	if err != nil {
		return err
	}

	s := &dhttp.ServerConfig{
		Handler: sm,
	}

	return s.Serve(ctx, listener)
}
