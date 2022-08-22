package entrypoint

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"sync/atomic"

	"github.com/datawire/dlib/dhttp"
	"github.com/datawire/dlib/dlog"
	snapshotTypes "github.com/emissary-ingress/emissary/v3/pkg/snapshot/v1"
)

// take the next port in the range of ambassador ports.
const ExternalSnapshotPort = 8005

// expose a scrubbed version of the current snapshot outside the pod
func externalSnapshotServer(ctx context.Context, snapshot *atomic.Value) error {
	mux := http.NewServeMux()
	mux.HandleFunc("/snapshot-external", func(w http.ResponseWriter, r *http.Request) {
		sanitizedSnap, err := sanitizeExternalSnapshot(ctx, snapshot.Load().([]byte), http.DefaultClient)
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

func sanitizeExternalSnapshot(ctx context.Context, rawSnapshot []byte, client *http.Client) ([]byte, error) {
	snapDecoded := snapshotTypes.Snapshot{}
	err := json.Unmarshal(rawSnapshot, &snapDecoded)
	if err != nil {
		return nil, err
	}
	err = snapDecoded.Sanitize()
	if err != nil {
		return nil, err
	}
	isEdgeStack, err := IsEdgeStack()
	if err != nil {
		return nil, err
	}
	if snapDecoded.AmbassadorMeta != nil && isEdgeStack {
		sidecarProcessInfoUrl := fmt.Sprintf("%s/process-info/", GetSidecarHost())
		dlog.Debugf(ctx, "loading sidecar process-info using [%s]...", sidecarProcessInfoUrl)
		resp, err := client.Get(sidecarProcessInfoUrl)
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

	return json.Marshal(snapDecoded)
}
