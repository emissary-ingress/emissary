package server

import (
	"log"
	"net/http"
	"github.com/gorilla/mux"
	"github.com/datawire/apro/cmd/dev-portal-server/kubernetes"
)

type server struct {
	router *mux.Router
	k8sstore kubernetes.ServiceStore
}

func (s *server) ServeHTTP() {
	log.Fatal(http.ListenAndServe(":8080", s.router));
}

func (s *server) handleOpenAPIListing() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// use thing
	}
}

func (s *server) handleOpenAPIGet() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// use thing
	}
}

func (s *server) handleOpenAPIUpdate() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// use thing
	}
}


func NewServer() *server {
	router := mux.NewRouter()
	s := &server{router: router, k8sstore: kubernetes.NewInMemoryStore()}

	// Static website: XXX figure out how to get static files. or maybe just
	// hardcode raw versions on github for now?
	router.Handle("/", http.FileServer(http.Dir("/tmp"))).Methods("GET")

	// Read-only API, requires less access control and may be exposed
	// publicly:
	// List services:
	router.
		HandleFunc("/openapi/services", s.handleOpenAPIListing()).
		Methods("GET")
	router.
		HandleFunc("/openapi/services/{id}", s.handleOpenAPIGet()).
		Methods("GET")

	// Write API, needs access control at some point:
	router.
		HandleFunc("/openapi/services", s.handleOpenAPIUpdate()).
		Methods("POST")

	return s
}
