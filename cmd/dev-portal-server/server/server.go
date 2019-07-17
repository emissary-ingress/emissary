package server

import (
	"encoding/json"
	"io/ioutil"
	"net/http"

	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"

	"github.com/datawire/apro/cmd/dev-portal-server/content"
	"github.com/datawire/apro/cmd/dev-portal-server/kubernetes"
	"github.com/datawire/apro/cmd/dev-portal-server/openapi"
	"github.com/datawire/apro/lib/logging"
)

type Server struct {
	router   *mux.Router
	content  *content.Content
	K8sStore kubernetes.ServiceStore
}

func (s *Server) knownServices() []kubernetes.Service {
	serviceMap := s.K8sStore.List()
	knownServices := make([]kubernetes.Service, len(serviceMap))
	i := 0
	for k := range serviceMap {
		knownServices[i] = k
		i++
	}
	return knownServices
}

func (s *Server) getServiceAdd() AddServiceFunc {
	return func(
		service kubernetes.Service, baseURL string, prefix string,
		openAPIDoc []byte) {
		hasDoc := (openAPIDoc != nil)
		var doc *openapi.OpenAPIDoc = nil
		if hasDoc {
			doc = openapi.NewOpenAPI(openAPIDoc, baseURL, prefix)
		}
		s.K8sStore.Set(
			service, kubernetes.ServiceMetadata{
				Prefix: prefix, BaseURL: baseURL,
				HasDoc: hasDoc, Doc: doc})
	}
}

func (s *Server) getServiceDelete() DeleteServiceFunc {
	return func(service kubernetes.Service) {
		s.K8sStore.Delete(service)
	}
}

func (s *Server) ServeHTTP() {
	log.Fatal(http.ListenAndServe("0.0.0.0:8680", s.router))
}

type openAPIListing struct {
	ServiceName      string `json:"service_name"`
	ServiceNamespace string `json:"service_namespace"`
	Prefix           string `json:"routing_prefix"`
	BaseURL          string `json:"routing_base_url"`
	HasDoc           bool   `json:"has_doc"`
}

func (s *Server) handleOpenAPIListing() http.HandlerFunc {
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

func (s *Server) handleOpenAPIGet() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		metadata := s.K8sStore.Get(kubernetes.Service{
			Name: vars["service"], Namespace: vars["namespace"]},
			true)
		if metadata == nil {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		js := metadata.Doc.JSON.EncodeJSON()
		w.Header().Set("Content-Type", "application/json")
		w.Write(js)
	}
}

type openAPIUpdate struct {
	ServiceName      string      `json:"service_name"`
	ServiceNamespace string      `json:"service_namespace"`
	Prefix           string      `json:"routing_prefix"`
	BaseURL          string      `json:"routing_base_url"`
	Doc              interface{} `json:"openapi_doc"`
}

func (s *Server) handleOpenAPIUpdate() http.HandlerFunc {
	addService := s.getServiceAdd()

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
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		hasdoc := (msg.Doc != nil)
		var buf []byte
		if hasdoc {
			buf, _ = json.Marshal(msg.Doc)
		} else {
			buf = nil
		}
		addService(
			kubernetes.Service{
				Name: msg.ServiceName, Namespace: msg.ServiceNamespace},
			msg.BaseURL, msg.Prefix, buf)
	}
}

type contentVars struct {
	S      *Server
	Ctx    string
	Prefix string
	Pages  []string
	Rq     map[string]string
}

func (s *Server) vars(r *http.Request, context, prefix string) content.ContentVars {
	return &contentVars{
		S:      s,
		Ctx:    context,
		Prefix: prefix,
		Rq:     mux.Vars(r),
	}
}

func (vars *contentVars) SetPages(pages []string) {
	vars.Pages = pages
}

func (vars *contentVars) CurrentPage() (page string) {
	page, _ = vars.Rq["page"]
	return
}

func (vars *contentVars) CurrentNamespace() (page string) {
	page, _ = vars.Rq["namespace"]
	return
}

func (vars *contentVars) CurrentService() (page string) {
	page, _ = vars.Rq["service"]
	return
}

func (s *Server) handleHTML(context string, prefix string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := s.vars(r, context, prefix)
		tmpl, err := s.content.Get(vars)
		if err != nil {
			log.Fatal(err)
		}
		w.Header().Set("Content-Type", "text/html")
		err = tmpl.Lookup("///layout").Execute(w, vars)
		if err != nil {
			log.Fatal(err)
		}
	}
}

func (s *Server) handleStatic() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		content, err := s.content.GetStatic(r.URL.Path)
		if err != nil {
			log.Warn(err)
			w.WriteHeader(http.StatusNotFound)
			return
		}
		http.ServeContent(w, r, content.Name, content.Modtime, content.Data)
		content.Close()
	}
}

// Create a new HTTP server instance.
//
// TODO The URL scheme exposes Service names and K8s namespace names, which is
// perhaps a security risk, and more broadly might be embarrassing for some
// organizations. So might want some better URL scheme.
func NewServer(content *content.Content) *Server {
	router := mux.NewRouter()
	router.Use(logging.LoggingMiddleware)

	s := &Server{
		router:   router,
		content:  content,
		K8sStore: kubernetes.NewInMemoryStore()}

	// TODO in a later design iteration, we would serve static HTML, and
	// have Javascript UI that queries the API endpoints. for this
	// iteration, just doing it server-side.
	router.HandleFunc("/", s.handleHTML("landing", ""))
	router.PathPrefix("/static/").HandlerFunc(s.handleStatic())
	router.HandleFunc("/page/{page}", s.handleHTML("page", "../"))
	router.HandleFunc("/doc/{namespace}/{service}", s.handleHTML("doc", "../../"))

	// *** Read-only API, requires less access control and may be exposed ***
	// publicly:
	// List services:
	router.
		HandleFunc("/openapi/services", s.handleOpenAPIListing()).
		Methods("GET")
	// Return the OpenAPI JSON:
	router.
		HandleFunc(
			"/openapi/services/{namespace}/{service}/openapi.json",
			s.handleOpenAPIGet()).
		Methods("GET")

	// *** Write API, needs access control at some point ***
	// Set information about new service:
	router.
		HandleFunc("/openapi/services", s.handleOpenAPIUpdate()).
		Methods("POST")

	return s
}
