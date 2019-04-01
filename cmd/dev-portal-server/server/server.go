package server

import (
	"github.com/datawire/apro/cmd/dev-portal-server/kubernetes"
	"github.com/gorilla/mux"
	"log"
	"net/http"
)

type server struct {
	router   *mux.Router
	K8sStore kubernetes.ServiceStore
}

func (s *server) ServeHTTP() {
	log.Fatal(http.ListenAndServe(":8080", s.router))
}

type openAPIListing struct {
	ServiceName      string `json:"service_name"`
	ServiceNamespace string `json:"service_namespace"`
	Prefix           string `json:"routing_prefix"`
	HasDoc           bool   `json:"has_doc"`
}

func (s *server) handleOpenAPIListing() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		result := make([]openAPIListing, 0)
		for service, metadata := range s.K8sStore.list() {
			result = append(result, openAPIListing{
				ServiceName:      service.Name,
				ServiceNamespace: service.Namespace,
				Prefix:           metadata.Prefix,
				HasDoc:           metadata.HasDoc,
			})
		}

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
	s := &server{router: router, K8sStore: kubernetes.NewInMemoryStore()}

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
