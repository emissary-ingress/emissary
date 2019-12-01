package watt

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"sync"

	"github.com/datawire/ambassador/pkg/dlog"
	"github.com/pkg/errors"

	"github.com/datawire/apro/cmd/amb-sidecar/filters/handler/middleware"
)

// The part where you have to think about locking //////////////////////////////

type SnapshotStore struct {
	httpClient *http.Client

	lock sync.RWMutex
	// things guarded by 'lock'
	closed      bool
	snapshot    Snapshot
	subscribers []chan<- Snapshot
}

func NewSnapshotStore(httpClient *http.Client) *SnapshotStore {
	return &SnapshotStore{
		httpClient: httpClient,
	}
}

func (ss *SnapshotStore) Get() Snapshot {
	ss.lock.RLock()
	defer ss.lock.RUnlock()
	return ss.snapshot
}

func (ss *SnapshotStore) Set(s Snapshot) {
	ss.lock.Lock()
	defer ss.lock.Unlock()

	if ss.closed {
		// block forever
		ss.lock.Unlock()
		select {}
	}

	ss.snapshot = s
	for _, subscriber := range ss.subscribers {
		subscriber <- s
	}
}

func (ss *SnapshotStore) makeSubscriberCh() <-chan Snapshot {
	ss.lock.Lock()
	defer ss.lock.Unlock()

	ret := make(chan Snapshot)
	if ss.closed {
		close(ret)
	} else {
		ss.subscribers = append(ss.subscribers, ret)
	}
	return ret
}

func (ss *SnapshotStore) Close() {
	ss.lock.Lock()
	defer ss.lock.Unlock()

	ss.closed = true
	for _, subscriber := range ss.subscribers {
		close(subscriber)
	}
	ss.subscribers = nil
}

// The part where you don't have to think about locking ////////////////////////

func (ss *SnapshotStore) Subscribe() <-chan Snapshot {
	upstream := ss.makeSubscriberCh()
	downstream := make(chan Snapshot)

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

func (ss *SnapshotStore) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// POST localhost:8500/_internal/v0/watt?url=...

	if r.Method != http.MethodPost {
		middleware.ServeErrorResponse(w, r.Context(), http.StatusMethodNotAllowed,
			errors.New("method not allowed"), nil)
		return
	}

	logger := dlog.GetLogger(r.Context())

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
