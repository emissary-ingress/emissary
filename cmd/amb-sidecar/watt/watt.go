package watt

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"sync"

	"github.com/pkg/errors"

	"github.com/datawire/apro/cmd/amb-sidecar/filters/handler/middleware"
)

type SnapshotStore struct {
	httpClient *http.Client

	lock sync.RWMutex

	snapshot Snapshot

	subscribers []chan<- Snapshot
}

func (ss *SnapshotStore) Get() Snapshot {
	ss.lock.RLock()
	defer ss.lock.RUnlock()
	return ss.snapshot
}

func (ss *SnapshotStore) Set(s Snapshot) {
	ss.lock.Lock()
	defer ss.lock.Unlock()

	ss.snapshot = s
	for _, subscriber := range ss.subscribers {
		subscriber <- s
	}
}

func (ss *SnapshotStore) Subscribe() <-chan Snapshot {
	ss.lock.Lock()
	defer ss.lock.Unlock()

	upstream := make(chan Snapshot)
	downstream := make(chan Snapshot)

	ss.subscribers = append(ss.subscribers, upstream)
	go coalesce(upstream, downstream)
	return downstream
}

func coalesce(upstream <-chan Snapshot, downstream chan<- Snapshot) {
do_read:
	item, ok := <-upstream
did_read:
	if !ok {
		close(downstream)
		return
	}
	select {
	case downstream <- item:
		goto do_read
	case item, ok = <-upstream:
		goto did_read
	}
}

func (ss *SnapshotStore) Close() {
	ss.lock.Lock()
	for _, subscriber := range ss.subscribers {
		close(subscriber)
	}
	ss.subscribers = nil
}

func NewSnapshotStore(httpClient *http.Client) *SnapshotStore {
	return &SnapshotStore{
		httpClient: httpClient,
	}
}

func (ss *SnapshotStore) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// POST localhost:8500/_internal/v0/watt?url=...

	if r.Method != http.MethodPost {
		middleware.ServeErrorResponse(w, r.Context(), http.StatusMethodNotAllowed,
			errors.New("method not allowed"), nil)
		return
	}

	logger := middleware.GetLogger(r.Context())

	_, push := r.URL.Query()["push"]

	var bodyBytes []byte
	var err error

	if push {
		logger.Debugf("loading WATT snapshot from body")
		bodyBytes, err = ioutil.ReadAll(r.Body)
		if err != nil {
			middleware.ServeErrorResponse(w, r.Context(), http.StatusBadRequest,
				err, nil)
			return
		}
	} else {
		logger.Debugf("loading WATT snapshot from %q", r.FormValue("url"))
		resp, err := ss.httpClient.Get(r.FormValue("url"))
		if err != nil {
			middleware.ServeErrorResponse(w, r.Context(), http.StatusBadRequest,
				err, nil)
			return
		}
		defer resp.Body.Close()
		bodyBytes, err = ioutil.ReadAll(resp.Body)
		if err != nil {
			middleware.ServeErrorResponse(w, r.Context(), http.StatusBadRequest,
				err, nil)
			return
		}
	}

	var snapshot Snapshot
	snapshot.Raw = bodyBytes
	if err := json.Unmarshal(bodyBytes, &snapshot); err != nil {
		middleware.ServeErrorResponse(w, r.Context(), http.StatusBadRequest,
			err, nil)
		return
	}

	ss.Set(snapshot)
}
