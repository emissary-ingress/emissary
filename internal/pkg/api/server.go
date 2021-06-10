package api

import (
	"context"
	"encoding/json"
	"log"
	"net"
	"net/http"
	"os"

	"github.com/datawire/ambassador/v2/internal/pkg/dns"
	"github.com/datawire/ambassador/v2/internal/pkg/interceptor"
	"github.com/datawire/ambassador/v2/internal/pkg/route"

	"github.com/datawire/dlib/dhttp"
)

type APIServer struct {
	listener net.Listener
	config   dhttp.ServerConfig

	stop context.CancelFunc
	done <-chan error
}

func NewAPIServer(iceptor *interceptor.Interceptor) (*APIServer, error) {
	handler := http.NewServeMux()
	tables := "/api/tables/"
	handler.HandleFunc(tables, func(w http.ResponseWriter, r *http.Request) {
		table := r.URL.Path[len(tables):]

		switch r.Method {
		case http.MethodGet:
			result := iceptor.Render(table)
			if result == "" {
				http.NotFound(w, r)
			} else {
				w.Write(append([]byte(result), '\n'))
			}
		case http.MethodPost:
			d := json.NewDecoder(r.Body)
			var table []route.Table
			err := d.Decode(&table)
			if err != nil {
				http.Error(w, err.Error(), 400)
			} else {
				for _, t := range table {
					iceptor.Update(t)
				}
				dns.Flush()
			}
		case http.MethodDelete:
			iceptor.Delete(table)
		}
	})
	handler.HandleFunc("/api/search", func(w http.ResponseWriter, r *http.Request) {
		var paths []string
		switch r.Method {
		case http.MethodGet:
			paths = iceptor.GetSearchPath()
			result, err := json.Marshal(paths)
			if err != nil {
				panic(err)
			} else {
				w.Write(result)
			}
		case http.MethodPost:
			d := json.NewDecoder(r.Body)
			err := d.Decode(&paths)
			if err != nil {
				http.Error(w, err.Error(), 400)
			} else {
				iceptor.SetSearchPath(paths)
			}
		}
	})
	handler.HandleFunc("/api/shutdown", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Goodbye!\n"))
		p, err := os.FindProcess(os.Getpid())
		if err != nil {
			panic(err)
		}
		p.Signal(os.Interrupt)
	})

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, err
	}

	return &APIServer{
		listener: ln,
		config: dhttp.ServerConfig{
			Handler: handler,
		},
	}, nil
}

func (a *APIServer) Port() string {
	_, port, err := net.SplitHostPort(a.listener.Addr().String())
	if err != nil {
		panic(err)
	}
	return port
}

func (a *APIServer) Start() {
	ctx, cancel := context.WithCancel(context.TODO())
	ch := make(chan error)
	a.stop = cancel
	a.done = ch
	go func() {
		ch <- a.config.Serve(ctx, a.listener)
		close(ch)
	}()
}

func (a *APIServer) Stop() {
	a.stop()
	if err := <-a.done; err != nil {
		log.Printf("API Server: %v", err)
	}
}
