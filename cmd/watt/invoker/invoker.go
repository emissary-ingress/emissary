package invoker

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"

	"github.com/datawire/ambassador/v2/pkg/supervisor"
	"github.com/datawire/ambassador/v2/pkg/tpu"
	"github.com/datawire/dlib/dhttp"
)

type invoker struct {
	Snapshots          chan string
	mux                sync.Mutex
	invokedSnapshots   map[int]string
	id                 int
	notify             []string
	apiServerAuthority string

	// This stores the latest snapshot, but we don't assign an id
	// unless/until we invoke... some of these will be discarded
	// by the rate limiting/coalescing logic
	latestSnapshot string
	process        *supervisor.Process
}

func NewInvoker(addr string, notify []string) *invoker {
	return &invoker{
		Snapshots:          make(chan string),
		invokedSnapshots:   make(map[int]string),
		notify:             notify,
		apiServerAuthority: addr,
	}
}

func (a *invoker) Work(p *supervisor.Process) error {
	// The general strategy here is:
	//
	// 1. Be continuously reading all available snapshots from
	//    a.Snapshots and store them in the potentialSnapshot
	//    variable. This means at any given point (modulo caveats
	//    below), the potentialSnapshot variable will have the
	//    latest and greatest snapshot available.
	//
	// 2. At the same time, whenever there is capacity to write
	//    down the invoking channel, we send potentialSnapshot to
	//    be invoked.
	//
	//    The anonymous goroutine below will be constantly reading
	//    from the invoking channel and performing a blocking
	//    a.invoke(). This means that we can only *write* to the
	//    invoking channel when we are not currently processing a
	//    snapshot, but when that happens, we will still read from
	//    a.Snapshots and update potentialSnapshot.
	//
	// There are two caveats to the above:
	//
	// 1. At startup, we don't yet have a snapshot to write, but
	//    we're not invoking anything, so we will try to write
	//    something down the invoking channel. To cope with this,
	//    the invoking goroutine will ignore snapshots that
	//    consist of the empty string.
	//
	// 2. If we process a snapshot quickly, or if there aren't new
	//    snapshots available, then we end up busy looping and
	//    sending the same potentialSnapshot value down the
	//    invoking channel multiple times. To cope with this,
	//    whenever we have successfully written to the invoking
	//    channel, we do a *blocking* read of the next snapshot
	//    from a.Snapshots.

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
			a.process.Debugf("deleting snapshot %d", k)
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
		k := tpu.NewKeeper("notify", fmt.Sprintf("%s http://%s/snapshots/%d", n, a.apiServerAuthority, id))
		k.Limit = 1
		k.Start()
		k.Wait()
	}
}

type apiServer struct {
	listenNetwork string
	listenAddress string
	invoker       *invoker
}

type APIServer interface {
	Work(*supervisor.Process) error
}

func NewAPIServer(net, addr string, invoker *invoker) APIServer {
	return &apiServer{
		listenNetwork: net,
		listenAddress: addr,
		invoker:       invoker,
	}
}

func (s *apiServer) Work(p *supervisor.Process) error {
	mux := http.NewServeMux()
	mux.HandleFunc("/snapshots/", func(w http.ResponseWriter, r *http.Request) {
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

	listener, err := net.Listen(s.listenNetwork, s.listenAddress)
	if err != nil {
		return err
	}
	p.Ready()
	p.Logf("snapshot server listening on: %s:%s", s.listenNetwork, s.listenAddress)
	srv := &dhttp.ServerConfig{
		Handler: mux,
	}
	ctx, cancel := context.WithCancel(p.Context())
	return p.DoClean(
		func() error { return srv.Serve(ctx, listener) },
		func() error { cancel(); return nil })
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
