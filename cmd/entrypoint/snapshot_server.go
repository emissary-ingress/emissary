package entrypoint

import (
	"context"
	"net/http"
	"sync/atomic"

	"github.com/datawire/dlib/dutil"
)

func snapshotServer(ctx context.Context, snapshot *atomic.Value) error {
	mux := http.NewServeMux()
	mux.HandleFunc("/snapshot", func(w http.ResponseWriter, r *http.Request) {
		w.Write(snapshot.Load().([]byte))
	})

	s := &http.Server{
		Addr:    "localhost:9696",
		Handler: mux,
	}

	return dutil.ListenAndServeHTTPWithContext(ctx, s)
}
