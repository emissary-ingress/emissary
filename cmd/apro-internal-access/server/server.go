package server

import (
	"crypto/subtle"
	"net/http"

	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"

	"github.com/datawire/apro/cmd/apro-internal-access/secret"
	"github.com/datawire/apro/lib/logging"
)

type server struct {
	router *mux.Router
	secret string
}

func (s *server) handleAuthenticate() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		secret := r.Header.Get("X-Ambassador-Internal-Auth")
		if subtle.ConstantTimeCompare([]byte(secret), []byte(s.secret)) != 1 {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
		}
	}
}

func (s *server) ServeHTTP() {
	log.Fatal(http.ListenAndServe("0.0.0.0:5000", s.router))
}

func NewServer(sharedSecretPath string) *server {
	router := mux.NewRouter()
	router.Use(logging.LoggingMiddleware)

	s := &server{router: router, secret: secret.LoadSecret(sharedSecretPath)}
	router.PathPrefix("/extauth/").Handler(s.handleAuthenticate())
	return s
}
