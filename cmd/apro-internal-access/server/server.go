package server

import (
	"net/http"

	"github.com/datawire/apro/shared/logging"
	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"
)

type server struct {
	router *mux.Router
	secret string
}

func (s *server) handleAuthenticate() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		//w.Write(...)
	}
}

func (s *server) ServeHTTP() {
	log.Fatal(http.ListenAndServe("0.0.0.0:5000", s.router))
}

func NewServer() *server {
	router := mux.NewRouter()
	router.Use(logging.LoggingMiddleware)

	s := &server{router: router, secret: "XXX"}
	router.PathPrefix("/extauth/").Handler(s.handleAuthenticate())
	return s
}
