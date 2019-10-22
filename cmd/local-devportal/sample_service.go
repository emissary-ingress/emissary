package main

import (
	"fmt"
	"net/http"

	"github.com/Jeffail/gabs"
	"github.com/gorilla/mux"

	"github.com/datawire/apro/lib/logging"
)

type SampleService struct {
	router *mux.Router
	prefix string
}

func newSampleService(prefix string, swagger bool) *SampleService {
	router := mux.NewRouter()
	router.Use(logging.LoggingMiddleware)
	s := &SampleService{
		router: router,
		prefix: prefix,
	}

	if swagger {
		router.Path(fmt.Sprintf("%s/.ambassador-internal/openapi-docs", prefix)).HandlerFunc(s.swagger).Methods("GET")
	}
	router.Path(s.path("/foo")).HandlerFunc(s.foo)
	router.Path(s.path("/bar")).HandlerFunc(s.foo)

	return s
}

func (s *SampleService) ServeHTTP(rsp http.ResponseWriter, rq *http.Request) {
	s.router.ServeHTTP(rsp, rq)
}

func (s *SampleService) path(path string) string {
	return fmt.Sprintf("%s%s", s.prefix, path)
}

func (s *SampleService) swagger(rsp http.ResponseWriter, rq *http.Request) {
	blob := gabs.New()
	blob.Set("2.0", "swagger")
	blob.Set("0.3.4", "info", "version")
	blob.Set("An example Open API service", "info", "title")
	foo := s.path("/foo")
	blob.Array("paths", foo, "get", "tags")
	blob.ArrayAppend("foo", "paths", foo, "get", "tags")
	blob.ArrayAppend("bar", "paths", foo, "get", "tags")
	blob.Set("An example request with no markdown", "paths", foo, "get", "description")
	blob.Set("all is good", "paths", foo, "get", "responses", "200", "description")
	bar := s.path("/bar")
	blob.Array("paths", bar, "get", "tags")
	blob.ArrayAppend("foo", "paths", bar, "get", "tags")
	blob.Set("short summary", "paths", bar, "get", "summary")
	blob.Set("# Another example request\n\n with markdown", "paths", bar, "get", "description")
	blob.Set("all is good", "paths", bar, "get", "responses", "200", "description")

	rsp.Header().Add("Content-Type", "application/json")
	rsp.WriteHeader(200)
	rsp.Write([]byte(blob.String()))
}

func (s *SampleService) foo(rsp http.ResponseWriter, rq *http.Request) {
	blob := gabs.New()
	blob.Set("1.0", "version")
	blob.Set("missing", "logic")

	rsp.Header().Add("Content-Type", "application/json")
	rsp.WriteHeader(200)
	rsp.Write([]byte(blob.String()))
}
