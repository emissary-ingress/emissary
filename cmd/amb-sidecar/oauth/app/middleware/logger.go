package middleware

import (
	"net/http"

	"github.com/urfave/negroni"

	"github.com/datawire/apro/cmd/amb-sidecar/types"
)

// Logger middleware for logging HTTP calls.
type Logger struct {
	Logger types.Logger
}

func (l *Logger) ServeHTTP(rw http.ResponseWriter, r *http.Request, next http.HandlerFunc) {
	l.Logger.Debugf("[HTTP %s] %s %s", r.Method, r.Host, r.URL.Path)

	next(rw, r)

	status := rw.(negroni.ResponseWriter).Status()
	l.Logger.Debugf("[HTTP %v] %s %s", status, r.Method, r.URL.Path)
}
