package entrypoint

import (
	"context"
	"encoding/json"
	"net/http"
	"sync/atomic"

	snapshotTypes "github.com/datawire/ambassador/pkg/snapshot/v1"
	"github.com/datawire/dlib/dhttp"
)

// expose a scrubbed version of the current snapshot outside the pod
func externalSnapshotServer(ctx context.Context, snapshot *atomic.Value, ambClusterID string, ambID string, ambVersion string) error {
	mux := http.NewServeMux()
	mux.HandleFunc("/snapshot-external", func(w http.ResponseWriter, r *http.Request) {
		rawSnapshot := snapshot.Load().([]byte)
		snapDecoded := snapshotTypes.Snapshot{}
		err := json.Unmarshal(rawSnapshot, &snapDecoded)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		err = snapDecoded.Sanitize()
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		sanitizedSnap, err := json.Marshal(snapDecoded)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.Header().Set("x-ambassador-cluster-id", ambClusterID)
		w.Header().Set("x-ambassador-id", ambID)
		w.Header().Set("x-ambassador-version", ambVersion)
		w.Header().Set("content-type", "application/json")

		w.Write(sanitizedSnap)
	})

	s := &dhttp.ServerConfig{
		Handler: mux,
	}

	return s.ListenAndServe(ctx, ":9697")
}

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
