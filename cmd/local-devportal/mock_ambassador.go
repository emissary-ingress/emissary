package main

import (
	"fmt"
	"net/http"

	"github.com/Jeffail/gabs"
	"github.com/gorilla/mux"

	"github.com/datawire/apro/lib/logging"
)

type mockAmbassador struct {
	router   *mux.Router
	mappings []mapping
}

type mapping struct {
	location string
	name     string
	prefix   string
}

func newMockAmbassador() *mockAmbassador {
	router := mux.NewRouter()
	router.Use(logging.LoggingMiddleware)
	f := &mockAmbassador{
		router: router,
	}
	router.Path("/ambassador/v0/diag/").HandlerFunc(f.diagd)

	return f
}

func (f *mockAmbassador) ServeHTTP(rsp http.ResponseWriter, rq *http.Request) {
	f.router.ServeHTTP(rsp, rq)
}

func (f *mockAmbassador) diagd(rsp http.ResponseWriter, rq *http.Request) {
	blob := gabs.New()
	for _, m := range f.mappings {
		blob.Set(true, "groups", m.name, "_active")
		blob.Set("IRHTTPMappingGroup", "groups", m.name, "kind")
		blob.ArrayOfSize(1, "groups", m.name, "mappings")
		mapping, _ := blob.S("groups", m.name, "mappings").ObjectI(0)
		mapping.Set(m.prefix, "prefix")
		mapping.Set(m.name, "name")
		mapping.Set(m.location, "location")
		mapping.Set("", "rewrite")
	}
	rsp.Header().Add("Content-Type", "application/json")
	rsp.WriteHeader(200)
	rsp.Write([]byte(blob.String()))
}

func (f *mockAmbassador) addMapping(ns, name, prefix string, handler http.Handler) {
	f.router.PathPrefix(prefix).Handler(handler)
	if prefix[len(prefix)-1:] != "/" {
		prefix += "/"
	}
	f.mappings = append(f.mappings, mapping{
		location: fmt.Sprintf("%s.%s.cluster", name, ns),
		name:     name,
		prefix:   prefix,
	})
}
