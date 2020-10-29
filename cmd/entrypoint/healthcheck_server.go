package entrypoint

import (
	"context"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"time"

	"github.com/datawire/ambassador/pkg/acp"
)

func handleCheckAlive(ctx context.Context, w http.ResponseWriter, r *http.Request, ambwatch *acp.AmbassadorWatcher) {
	// The liveness check needs to explicitly try to talk to Envoy...
	ambwatch.FetchEnvoyStats(ctx)

	// ...then check if the watcher says we're alive.
	ok := ambwatch.IsAlive()

	if ok {
		w.Write([]byte("Ambassador is alive and well\n"))
	} else {
		http.Error(w, "Ambassador is not alive\n", http.StatusServiceUnavailable)
	}
}

func handleCheckReady(_ context.Context, w http.ResponseWriter, r *http.Request, ambwatch *acp.AmbassadorWatcher) {
	ok := ambwatch.IsReady()

	if ok {
		w.Write([]byte("Ambassador is ready and waiting\n"))
	} else {
		http.Error(w, "Ambassador is not ready\n", http.StatusServiceUnavailable)
	}
}

func healthCheckHandler(ctx context.Context, ambwatch *acp.AmbassadorWatcher) {
	// We need to do some HTTP stuff by hand to catch the readiness and liveness
	// checks here, but forward everything else to diagd.
	sm := http.NewServeMux()

	// Handle the liveness check and the readiness check directly, by handing them
	// off to our functions.
	sm.HandleFunc("/ambassador/v0/check_alive", func(w http.ResponseWriter, r *http.Request) {
		handleCheckAlive(ctx, w, r, ambwatch)
	})

	sm.HandleFunc("/ambassador/v0/check_ready", func(w http.ResponseWriter, r *http.Request) {
		handleCheckReady(ctx, w, r, ambwatch)
	})

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
		},
	}

	// Finally, use the reverseProxy to handle anything coming in on
	// the magic catchall path.
	sm.HandleFunc("/", reverseProxy.ServeHTTP)

	s := &http.Server{
		Addr:    "0.0.0.0:8877",
		Handler: sm,
	}

	// Given that, all that's left is to fire up a server using our
	// router.
	go func() {
		log.Println(s.ListenAndServe())
	}()

	// ...then wait for a shutdown signal.
	<-ctx.Done()

	tctx, tcancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer tcancel()

	err := s.Shutdown(tctx)

	if err != nil {
		panic(err)
	}
}
