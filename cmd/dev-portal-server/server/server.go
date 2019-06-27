package server

import (
	"encoding/json"
	"html/template"
	"io/ioutil"
	"net/http"

	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"

	"github.com/datawire/apro/cmd/dev-portal-server/kubernetes"
	"github.com/datawire/apro/cmd/dev-portal-server/openapi"
	"github.com/datawire/apro/lib/logging"
)

type server struct {
	router   *mux.Router
	K8sStore kubernetes.ServiceStore
}

func (s *server) knownServices() []kubernetes.Service {
	serviceMap := s.K8sStore.List()
	knownServices := make([]kubernetes.Service, len(serviceMap))
	i := 0
	for k := range serviceMap {
		knownServices[i] = k
		i++
	}
	return knownServices
}

func (s *server) getServiceAdd() AddServiceFunc {
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

func (s *server) getServiceDelete() DeleteServiceFunc {
	return func(service kubernetes.Service) {
		s.K8sStore.Delete(service)
	}
}

func (s *server) ServeHTTP() {
	log.Fatal(http.ListenAndServe("0.0.0.0:8680", s.router))
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

func (s *server) handleOpenAPIUpdate() http.HandlerFunc {
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
			http.Error(w, err.Error(), http.StatusInternalServerError)
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

func (s *server) handleIndexHTML() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		funcMap := template.FuncMap{
			// The name "inc" is what the function will be called in the template text.
			"isEven": func(i int) bool {
				return i%2 == 0
			},

			"isOdd": func(i int) bool {
				return i%2 != 0
			},
		}

		tmpl, err := template.New("index").Funcs(funcMap).Parse(`
<html lang="utf-8">
<head>
	<title>Developer Portal</title>

	<!-- Bootstrap core CSS -->
	<link rel="stylesheet" href="https://stackpath.bootstrapcdn.com/bootstrap/4.1.1/css/bootstrap.min.css" integrity="sha384-WskhaSGFgHYWDcbwN70/dfYBj47jz9qbsMId/iRN3ewGhXQFZCSftd1LZCfmhktB" crossorigin="anonymous">

	<!-- Custom styles for this template -->
	<link href="https://getbootstrap.com/docs/4.0/examples/grid/grid.css" rel="stylesheet">
</head>
<body>
<div class="container">
	<h1>Developer Portal</h1>
	<div class="row">
		<div class="col-12">
			Available Services
			<div class="row">
				<div class="col-12">
					<table cellpadding="2em" width="100%">
						<thead>
							<tr>
								<td><b>Service Name</b></td>
								<td><b>Swagger URL</b></td>
							</tr>
						</thead>
						<tbody>
							{{range $i, $element := .K8sStore.Slice }}
							{{if isEven $i }}
							<tr style="background: rgba(86,61,124,.05);">
							{{else}}
							<tr>
							{{end}}
								<td>
									<samp>{{$element.Service.Namespace}}.{{$element.Service.Name}}</samp>
								</td>
								<td>
									{{if $element.Metadata.HasDoc}}
    								<a href="doc/{{$element.Service.Namespace}}/{{$element.Service.Name}}"><code>API Documentation</code></a>
									{{else}}
									<code><span style="color:red">No API Documentation</span></code>
    								{{end}}
								</td>
							</tr>
							{{end}}
						</tbody>
					</table>
				</div>
			</div>
		</div>
	</div>
</div>
</body>
</html>
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
    url: "../../openapi/services/{{.namespace}}/{{.name}}/openapi.json",
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
// perhaps a security risk, and more broadly might be embarrassing for some
// organizations. So might want some better URL scheme.
func NewServer() *server {
	router := mux.NewRouter()
	router.Use(logging.LoggingMiddleware)

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
