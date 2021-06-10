package entrypoint

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync/atomic"

	snapshotTypes "github.com/datawire/ambassador/v2/pkg/snapshot/v1"
	"github.com/datawire/dlib/dhttp"
)

// take the next port in the range of ambassador ports.
const ExternalSnapshotPort = 8005

// expose a scrubbed version of the current snapshot outside the pod
func externalSnapshotServer(ctx context.Context, snapshot *atomic.Value) error {
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
		w.Header().Set("content-type", "application/json")

		w.Write(sanitizedSnap)
	})

	s := &dhttp.ServerConfig{
		Handler: mux,
	}

	return s.ListenAndServe(ctx, fmt.Sprintf(":%d", ExternalSnapshotPort))
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
