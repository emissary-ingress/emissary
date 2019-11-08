package server

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/oxtoacart/bpool"
	log "github.com/sirupsen/logrus"

	"github.com/datawire/apro/cmd/amb-sidecar/devportal/content"
	"github.com/datawire/apro/cmd/amb-sidecar/devportal/openapi"
	"github.com/datawire/apro/cmd/amb-sidecar/limiter"
	"github.com/datawire/apro/lib/licensekeys"
	"github.com/datawire/apro/lib/logging"
)

type Server struct {
	router       *mux.Router
	content      *content.Content
	limiter      limiter.Limiter
	climiter     limiter.CountLimiter
	serviceStore *inMemoryStore

	pool *bpool.BufferPool

	prefix string
}

func (s *Server) KnownServices() []Service {
	serviceMap := s.serviceStore.List()
	knownServices := make([]Service, len(serviceMap))
	i := 0
	for k := range serviceMap {
		knownServices[i] = k
		i++
	}
	return knownServices
}

// AddService implements ServiceStore.
func (s *Server) AddService(service Service, baseURL string, prefix string, openAPIDoc []byte) error {
	hasDoc := (openAPIDoc != nil)
	var doc *openapi.OpenAPIDoc = nil
	if hasDoc {
		doc = openapi.NewOpenAPI(openAPIDoc, baseURL, prefix)
	}
	return s.serviceStore.Set(
		service, ServiceMetadata{
			Prefix: prefix, BaseURL: baseURL,
			HasDoc: hasDoc, Doc: doc})
}

// DeleteService implements ServiceStore.
func (s *Server) DeleteService(service Service) error {
	return s.serviceStore.Delete(service)
}

func (s *Server) Router() http.Handler {
	return s.router
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
		for service, metadata := range s.serviceStore.List() {
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
		metadata := s.serviceStore.Get(Service{
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
		err = s.AddService(
			Service{
				Name: msg.ServiceName, Namespace: msg.ServiceNamespace},
			msg.BaseURL, msg.Prefix, buf)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
	}
}

type contentVars struct {
	S      ServerView
	Ctx    string
	Prefix string
	Pages  []string
	Rq     map[string]string
}

type ServerView interface {
	K8sStore() ServiceStoreView
}

type ServiceStoreView interface {
	Slice() []ServiceRecord
}

func (s *Server) K8sStore() ServiceStoreView {
	return s.serviceStore
}

func (s *Server) vars(r *http.Request, context string) content.ContentVars {
	return &contentVars{
		S:      s,
		Ctx:    context,
		Prefix: s.prefix,
		Rq:     mux.Vars(r),
	}
}

func (vars *contentVars) SetPages(pages []string) {
	vars.Pages = pages
}

func (vars *contentVars) CurrentPage() (page string) {
	page = vars.Rq["page"]
	return
}

func (vars *contentVars) CurrentNamespace() (page string) {
	page = vars.Rq["namespace"]
	return
}

func (vars *contentVars) CurrentService() (page string) {
	page = vars.Rq["service"]
	return
}

func (s *Server) handleHTML(context string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := s.vars(r, context)
		tmpl, err := s.content.Get(vars)
		if err != nil {
			log.Warn(s.content.Source(), err)
			w.WriteHeader(http.StatusInternalServerError)
			io.WriteString(w, err.Error())
			io.WriteString(w, "\n\n")
			io.WriteString(w, s.content.Source().String())
			return
		}
		buffer := s.pool.Get()
		defer s.pool.Put(buffer)
		err = tmpl.Lookup("///layout").Execute(buffer, vars)
		if err != nil {
			log.Warn(err)
			w.WriteHeader(http.StatusInternalServerError)
			io.WriteString(w, err.Error())
			return
		}
		w.Header().Set("Content-Type", "text/html")
		w.Header().Set("Content-Length",
			fmt.Sprintf("%d", buffer.Len()))
		buffer.WriteTo(w)
	}
}

func (s *Server) handleStatic(docroot string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		content, err := s.content.GetStatic(r.URL.Path[len(docroot):])
		if err != nil {
			log.Warn(err)
			w.WriteHeader(http.StatusNotFound)
			return
		}
		http.ServeContent(w, r, content.Name, content.Modtime, content.Data)
		content.Close()
	}
}

func (s *Server) Init(fetcher MappingSubscriptions) {
	fetcher.SubscribeMappingObserver("ambassador-pro-devportal", func(prefix, rewrite string) bool {
		prefix += "/"
		log.WithFields(log.Fields{
			"oldPrefix": s.prefix,
			"prefix":    prefix,
			"rewrite":   rewrite,
		}).Info("Prefix detected from ambassador-pro-devportal")
		s.prefix = prefix
		return true
	})
	fetcher.SubscribeMappingObserver("ambassador-pro-devportal-api", func(prefix, rewrite string) bool {
		prefix += "/"
		log.WithFields(log.Fields{
			"oldPrefix": s.prefix,
			"prefix":    prefix,
			"rewrite":   rewrite,
		}).Info("API prefix detected from ambassador-pro-devportal-api (TODO)")
		return true
	})
}

// Create a new HTTP server instance.
//
// TODO The URL scheme exposes Service names and K8s namespace names, which is
// perhaps a security risk, and more broadly might be embarrassing for some
// organizations. So might want some better URL scheme.
func NewServer(docroot string, content *content.Content, limiter limiter.Limiter) *Server {
	router := mux.NewRouter()
	router.Use(logging.LoggingMiddleware)

	// Error should never be set due to hardcoded enums
	// but if it is make it break hard.
	climiter, err := limiter.CreateCountLimiter(&licensekeys.LimitDevPortalServices)
	if err != nil {
		return nil
	}

	root := docroot + "/"
	s := &Server{
		router:       router,
		limiter:      limiter,
		climiter:     climiter,
		serviceStore: newInMemoryStore(climiter, limiter),
		content:      content,
		pool:         bpool.NewBufferPool(64),
		prefix:       root,
	}

	// TODO in a later design iteration, we would serve static HTML, and
	// have Javascript UI that queries the API endpoints. for this
	// iteration, just doing it server-side.
	router.
		HandleFunc(docroot+"/", s.handleHTML("landing")).
		Methods("GET")
	router.
		PathPrefix(docroot + "/assets/").HandlerFunc(s.handleStatic(docroot)).
		Methods("GET")
	router.
		PathPrefix(docroot + "/styles/").HandlerFunc(s.handleStatic(docroot)).
		Methods("GET")
	router.
		HandleFunc(docroot+"/page/{page}", s.handleHTML("page")).
		Methods("GET")
	router.
		HandleFunc(docroot+"/doc/{namespace}/{service}", s.handleHTML("doc")).
		Methods("GET")

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
