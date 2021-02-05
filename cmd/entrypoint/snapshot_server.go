package entrypoint

import (
	"context"
	"net/http"
	"sync/atomic"

	"github.com/datawire/dlib/dhttp"
)

func snapshotServer(ctx context.Context, snapshot *atomic.Value) error {
	mux := http.NewServeMux()
	mux.HandleFunc("/snapshot", func(w http.ResponseWriter, r *http.Request) {
		w.Write(snapshot.Load().([]byte))
	})

	s := &dhttp.ServerConfig{
		Handler: mux,
	}

	return s.ListenAndServe(ctx, "localhost:9696")
}
