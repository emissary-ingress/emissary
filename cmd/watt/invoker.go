package main

import (
	"fmt"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"

	"github.com/datawire/ambassador/pkg/supervisor"
	"github.com/datawire/ambassador/pkg/tpu"
)

type invoker struct {
	Snapshots        chan string
	mux              sync.Mutex
	invokedSnapshots map[int]string
	id               int
	notify           []string
	apiServerPort    int

	// This stores the latest snapshot, but we don't assign an id
	// unless/until we invoke... some of these will be discarded
	// by the rate limiting/coalescing logic
	latestSnapshot string
	process        *supervisor.Process
}

func NewInvoker(port int, notify []string) *invoker {
	return &invoker{
		Snapshots:        make(chan string),
		invokedSnapshots: make(map[int]string),
		notify:           notify,
		apiServerPort:    port,
	}
}

func (a *invoker) Work(p *supervisor.Process) error {
	a.process = p
	p.Ready()

	invoking := make(chan string)

	go func() {
		for snapshot := range invoking {
			// ignore empty snapshots to deal with the
			// corner case where we haven't yet received a
			// snapshot
			if snapshot != "" {
				a.latestSnapshot = snapshot
				a.invoke()
			}
		}
	}()

	potentialSnapshot := ""

	for {
		select {
		case potentialSnapshot = <-a.Snapshots:
			// if a new snapshot is available to be read,
			// and we can't write to the invoking channel,
			// then we will overwrite potentialSnapshot
			// with a newer snapshot
		case invoking <- potentialSnapshot:
			// if we aren't currently blocked in
			// a.invoke() then the above goroutine will be
			// reading from the invoking channel and we
			// will send the current potentialSnapshot
			// value over the invoking channel to be
			// processed

			select {
			case potentialSnapshot = <-a.Snapshots:
				// whenever we write a
				// potentialSnapshot to invoking, we
				// also need to do a synchronous read
				// from a.Snapshots to make sure we
				// won't ever invoke the same snapshot
				// twice
			case <-p.Shutdown():
				p.Logf("shutdown initiated")
				return nil
			}
		case <-p.Shutdown():
			p.Logf("shutdown initiated")
			return nil
		}
	}
}

func (a *invoker) storeSnapshot(snapshot string) int {
	a.mux.Lock()
	defer a.mux.Unlock()
	a.id += 1
	a.invokedSnapshots[a.id] = snapshot
	a.gcSnapshots()
	return a.id
}

func (a *invoker) gcSnapshots() {
	for k := range a.invokedSnapshots {
		if k <= a.id-10 {
			delete(a.invokedSnapshots, k)
			a.process.Logf("deleting snapshot %d", k)
		}
	}
}

func (a *invoker) getSnapshot(id int) string {
	a.mux.Lock()
	defer a.mux.Unlock()
	return a.invokedSnapshots[id]
}

func (a *invoker) getKeys() (result []int) {
	for i := range a.invokedSnapshots {
		result = append(result, i)
	}
	return
}

func (a *invoker) invoke() {
	id := a.storeSnapshot(a.latestSnapshot)
	for _, n := range a.notify {
		k := tpu.NewKeeper("notify", fmt.Sprintf("%s http://localhost:%d/snapshots/%d", n, a.apiServerPort, id))
		k.Limit = 1
		k.Start()
		k.Wait()
	}
}

type apiServer struct {
	port    int
	invoker *invoker
}

func (s *apiServer) Work(p *supervisor.Process) error {
	http.HandleFunc("/snapshots/", func(w http.ResponseWriter, r *http.Request) {
		relpath := strings.TrimPrefix(r.URL.Path, "/snapshots/")

		if relpath == "" {
			w.Header().Set("content-type", "text/html")
			if _, err := w.Write([]byte(s.index())); err != nil {
				p.Logf("write index error: %v", err)
			}
		} else {
			id, err := strconv.Atoi(relpath)
			if err != nil {
				http.Error(w, "ID is not an integer", http.StatusBadRequest)
				return
			}

			snapshot := s.invoker.getSnapshot(id)

			if snapshot == "" {
				w.WriteHeader(http.StatusNotFound)
				return
			}

			w.Header().Set("content-type", "application/json")
			if _, err := w.Write([]byte(snapshot)); err != nil {
				p.Logf("write snapshot error: %v", err)
			}
		}
	})

	listenHostAndPort := fmt.Sprintf(":%d", s.port)
	listener, err := net.Listen("tcp", listenHostAndPort)
	if err != nil {
		return err
	}
	p.Ready()
	p.Logf("snapshot server listening on: %s", listenHostAndPort)
	srv := &http.Server{
		Addr: listenHostAndPort,
	}
	return p.DoClean(func() error {
		err := srv.Serve(listener)
		if err == http.ErrServerClosed {
			return nil
		}
		return err
	},
		func() error {
			return srv.Shutdown(p.Context())
		})

}

func (s *apiServer) index() string {
	var result strings.Builder

	result.WriteString("<html><body><ul>\n")

	for _, v := range s.invoker.getKeys() {
		result.WriteString(fmt.Sprintf("  <li><a href=\"%d\">%d</a></li>\n", v, v))
	}

	result.WriteString("</ul></body></html>\n")

	return result.String()
}
