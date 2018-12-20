package middleware

import (
	"net/http"

	"github.com/sirupsen/logrus"
	"github.com/urfave/negroni"
)

// Logger middleware for logging HTTP calls.
type Logger struct {
	Logger *logrus.Entry
}

func (l *Logger) ServeHTTP(rw http.ResponseWriter, r *http.Request, next http.HandlerFunc) {
	l.Logger.Debugf("[HTTP %s] %s %s", r.Method, r.Host, r.URL.Path)

	next(rw, r)

	status := rw.(negroni.ResponseWriter).Status()
	if status > 499 && status < 599 {
		l.Logger.Debugf("[HTTP %v] %s %s", status, r.Method, r.URL.Path)
	} else {
		l.Logger.Debugf("[HTTP %v] %s %s", status, r.Method, r.URL.Path)
	}
}
