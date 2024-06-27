package entrypoint

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync/atomic"

	"github.com/datawire/dlib/dhttp"
	snapshotTypes "github.com/emissary-ingress/emissary/v3/pkg/snapshot/v1"
)

// take the next port in the range of ambassador ports.
const ExternalSnapshotPort = 8005

// expose a scrubbed version of the current snapshot outside the pod
func externalSnapshotServer(ctx context.Context, snapshot *atomic.Value) error {
	mux := http.NewServeMux()
	mux.HandleFunc("/snapshot-external", func(w http.ResponseWriter, r *http.Request) {
		sanitizedSnap, err := sanitizeExternalSnapshot(ctx, snapshot.Load().([]byte))
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.Header().Set("content-type", "application/json")
		_, _ = w.Write(sanitizedSnap)
	})

	s := &dhttp.ServerConfig{
		Handler: mux,
	}

	return s.ListenAndServe(ctx, fmt.Sprintf(":%d", ExternalSnapshotPort))
}

func snapshotServer(ctx context.Context, snapshot *atomic.Value) error {
	mux := http.NewServeMux()
	mux.HandleFunc("/snapshot", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(snapshot.Load().([]byte))
	})

	s := &dhttp.ServerConfig{
		Handler: mux,
	}

	return s.ListenAndServe(ctx, "localhost:9696")
}

func sanitizeExternalSnapshot(ctx context.Context, rawSnapshot []byte) ([]byte, error) {
	snapDecoded := snapshotTypes.Snapshot{}
	err := json.Unmarshal(rawSnapshot, &snapDecoded)
	if err != nil {
		return nil, err
	}
	err = snapDecoded.Sanitize()
	if err != nil {
		return nil, err
	}

	return json.Marshal(snapDecoded)
}
