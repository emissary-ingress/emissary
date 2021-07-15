package entrypoint

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"sync/atomic"

	snapshotTypes "github.com/datawire/ambassador/v2/pkg/snapshot/v1"
	"github.com/datawire/dlib/dhttp"
	"github.com/datawire/dlib/dlog"
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
		if snapDecoded.AmbassadorMeta != nil && IsEdgeStack() {
			sidecarProcessInfoUrl := fmt.Sprintf("%s/process-info/", GetSidecarHost())
			dlog.Debugf(ctx, "loading sidecard process-info using [%s]...", sidecarProcessInfoUrl)
			resp, err := http.DefaultClient.Get(sidecarProcessInfoUrl)
			if err != nil {
				dlog.Error(ctx, err.Error())
			} else {
				defer resp.Body.Close()
				if resp.StatusCode == 200 {
					bodyBytes, err := ioutil.ReadAll(resp.Body)
					if err != nil {
						dlog.Warnf(ctx, "error reading response body: %v", err)
					} else {
						snapDecoded.AmbassadorMeta.Sidecar = bodyBytes
					}
				} else {
					dlog.Warnf(ctx, "unexpected status code %v", resp.StatusCode)
				}
			}
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
