package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/datawire/apro/cmd/model-cluster-app/comments"
	"github.com/datawire/apro/cmd/model-cluster-app/events"
	"github.com/datawire/apro/cmd/model-cluster-app/health"
	"github.com/datawire/apro/cmd/model-cluster-app/posts"
	"github.com/datawire/apro/cmd/model-cluster-app/users"
	"github.com/datawire/apro/lib/util"
	"github.com/gorilla/mux"
)

// Version is inserted at build using --ldflags -X
var Version = "(unknown version)"

// home returns a simple HTTP handler function which writes a response.
func home(release string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		fail := func(msg string, err error) {
			log.Print(msg)
			log.Print(err)
			http.Error(w, http.StatusText(http.StatusServiceUnavailable), http.StatusServiceUnavailable)
		}
		client := util.NewHeaderPassingClient(r, []string{"moo"})
		resp, err := client.Get("http://localhost:8080/target")
		if err != nil {
			fail("get failed", err)
			return
		}
		defer resp.Body.Close()
		targetRes, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			fail("reading body failed", err)
			return
		}

		info := struct {
			Release   string `json:"release"`
			TargetRes string `json:"target_res"`
		}{
			release,
			string(targetRes),
		}

		body, err := json.Marshal(info)
		if err != nil {
			fail("could not encode info data", err)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write(body)
	}
}

func main() {
	log.Printf("Starting model service (version %s)", Version)

	// Hurry up and start the server so liveness checks can pass

	port := os.Getenv("PORT")
	if port == "" {
		log.Fatal("Port is not set.")
	}

	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt, syscall.SIGTERM)

	isReady := &atomic.Value{}
	isReady.Store(false)

	r := mux.NewRouter()
	r.HandleFunc("/home", home(Version)).Methods("GET")
	r.HandleFunc("/target", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(fmt.Sprintf("Hello World: %s\n", r.Header.Get("moo"))))
	})
	r.HandleFunc("/healthz", health.Healthz)
	r.HandleFunc("/readyz", health.Readyz(isReady))

	apiRouter := r.PathPrefix("/api/v1").Subrouter()
	users.RegisterHandlers(apiRouter)
	posts.RegisterHandlers(apiRouter)
	comments.RegisterHandlers(apiRouter)
	events.RegisterHandlers(apiRouter)

	server := &http.Server{
		Addr:         "127.0.0.1:" + port,
		Handler:      r,
		WriteTimeout: 15 * time.Second,
		ReadTimeout:  15 * time.Second,
	}

	shutdown := make(chan error)
	go func() {
		shutdown <- server.ListenAndServe()
	}()
	log.Print("Server is active.")

	// Now perform slow startup stuff
	// e.g., connect to a remote database
	// ...

	// Set ready flag so readiness handler returns success
	isReady.Store(true)
	log.Print("All services are ready.")

	// Wait for exit...
	select {
	case killSignal := <-interrupt:
		switch killSignal {
		case os.Interrupt:
			log.Print("Got SIGINT...")
		case syscall.SIGTERM:
			log.Print("Got SIGTERM...")
		}
	case err := <-shutdown:
		log.Printf("Got an error: %v", err)
	}

	log.Print("The service is shutting down...")
	server.Shutdown(context.Background())
	log.Print("Done")
}
