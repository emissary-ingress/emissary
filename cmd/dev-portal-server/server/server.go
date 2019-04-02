package server

import (
	"encoding/json"
	"github.com/datawire/apro/cmd/dev-portal-server/kubernetes"
	"github.com/datawire/apro/cmd/dev-portal-server/openapi"
	"github.com/gorilla/mux"
	"html/template"
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
			doc = openapi.NewOpenAPI(msg.Doc, msg.BaseURL, msg.Prefix)
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

func (s *server) handleIndexHTML() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tmpl, err := template.New("index").Parse(`
<h1>Available services</h1>
{{range $service, $metadata := .K8sStore.List }}
<p>
<strong>{{$service.Namespace}}/{{$service.Name}}</strong>
    {{if $metadata.HasDoc}}
    <a href="/doc/{{$service.Namespace}}/{{$service.Name}}">Docs</a>
    {{end}}
</p>
{{end}}
`)
		if err != nil {
			log.Fatal(err)
		}
		w.Header().Set("Content-Type", "text/html")
		err = tmpl.Execute(w, s)
		if err != nil {
			log.Fatal(err)
		}
	}
}

func (s *server) handleDocHTML() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tmpl, err := template.New("doc").Parse(`
<head>
<script src="https://unpkg.com/swagger-ui-dist@3/swagger-ui-bundle.js"></script>
<script src="https://unpkg.com/swagger-ui-dist@3/swagger-standalone-preset.js"></script>
<link rel="stylesheet" type="text/css" href="https://unpkg.com/swagger-ui-dist@3/swagger-ui.css" >
</head>
<body>
<span id="swagger-ui"></span>
<script>
const ui = SwaggerUIBundle({
    url: "/openapi/services/{{.namespace}}/{{.name}}/openapi.json",
    dom_id: '#swagger-ui',
    presets: [
      SwaggerUIBundle.presets.apis,
      SwaggerUIBundle.SwaggerUIStandalonePreset
    ],
  })
</script>
</body>
`)
		if err != nil {
			log.Fatal(err)
		}
		vars := mux.Vars(r)
		w.Header().Set("Content-Type", "text/html")
		err = tmpl.Execute(w, vars)
		if err != nil {
			log.Fatal(err)
		}
	}
}

// Create a new HTTP server instance.
//
// TODO The URL scheme exposes Service names and K8s namespace names, which is
// perhaps a security risk, and more broadly might be embarassing for some
// organizations. So might want some better URL scheme.
func NewServer() *server {
	router := mux.NewRouter()
	s := &server{router: router, K8sStore: kubernetes.NewInMemoryStore()}

	// TODO in a later design iteration, we would serve static HTML, and
	// have Javascript UI that queries the API endpoints. for this
	// iteration, just doing it server-side.
	router.HandleFunc("/", s.handleIndexHTML())
	router.HandleFunc("/doc/{namespace}/{name}", s.handleDocHTML())

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
