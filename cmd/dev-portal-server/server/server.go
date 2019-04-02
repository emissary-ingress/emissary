package server

import (
	"encoding/json"
	"github.com/datawire/apro/cmd/dev-portal-server/kubernetes"
	"github.com/datawire/apro/cmd/dev-portal-server/openapi"
	"github.com/gorilla/mux"
	"io/ioutil"
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
	BaseURL          string `json:"routing_base_url"`
	HasDoc           bool   `json:"has_doc"`
}

func (s *server) handleOpenAPIListing() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		result := make([]openAPIListing, 0)
		for service, metadata := range s.K8sStore.List() {
			result = append(result, openAPIListing{
				ServiceName:      service.Name,
				ServiceNamespace: service.Namespace,
				Prefix:           metadata.Prefix,
				BaseURL:          metadata.BaseURL,
				HasDoc:           metadata.HasDoc,
			})
		}
		js, err := json.Marshal(result)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.Write(js)
	}
}

func (s *server) handleOpenAPIGet() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		metadata := s.K8sStore.Get(kubernetes.Service{
			Name: vars["name"], Namespace: vars["namespace"]},
			true)
		js, err := json.Marshal(metadata.Doc.JSON.EncodeJSON())
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.Write(js)
	}
}

type openAPIUpdate struct {
	ServiceName      string      `json:"service_name"`
	ServiceNamespace string      `json:"service_namespace"`
	Prefix           string      `json:"routing_prefix"`
	BaseURL          string      `json:"routing_host"`
	Doc              interface{} `json:"openapi_doc"`
}

func (s *server) handleOpenAPIUpdate() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		b, err := ioutil.ReadAll(r.Body)
		defer r.Body.Close()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		var msg openAPIUpdate
		err = json.Unmarshal(b, &msg)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		hasdoc := (msg.Doc != nil)
		var doc *openapi.OpenAPIDoc
		if hasdoc {
			doc = openapi.NewOpenAPI(msg.Doc, msg.Prefix, msg.BaseURL)
		} else {
			doc = nil
		}
		s.K8sStore.Set(
			kubernetes.Service{
				Name: msg.ServiceName, Namespace: msg.ServiceNamespace},
			kubernetes.ServiceMetadata{
				Prefix: msg.Prefix, BaseURL: msg.BaseURL,
				HasDoc: hasdoc, Doc: doc})
	}
}

func NewServer() *server {
	router := mux.NewRouter()
	s := &server{router: router, K8sStore: kubernetes.NewInMemoryStore()}

	// Static website: XXX figure out how to get static files. or maybe just
	// hardcode raw versions on github for now?
	router.Handle("/", http.FileServer(http.Dir("/tmp"))).Methods("GET")

	// *** Read-only API, requires less access control and may be exposed ***
	// publicly:
	// List services:
	router.
		HandleFunc("/openapi/services", s.handleOpenAPIListing()).
		Methods("GET")
	// Return the OpenAPI JSON:
	router.
		HandleFunc(
			"/openapi/services/{namespace}/{name}/openapi.json",
			s.handleOpenAPIGet()).
		Methods("GET")

	// *** Write API, needs access control at some point ***
	// Set information about new service:
	router.
		HandleFunc("/openapi/services", s.handleOpenAPIUpdate()).
		Methods("POST")

	return s
}
