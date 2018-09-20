package middleware

import (
	"net/http"

	"github.com/sirupsen/logrus"
	"github.com/urfave/negroni"
)

// Logger middleware for logging HTTP calls.
type Logger struct {
	Logger *logrus.Logger
}

func (l *Logger) ServeHTTP(rw http.ResponseWriter, r *http.Request, next http.HandlerFunc) {
	l.Logger.Debugf("downstream: %s %s%s", r.Method, r.Host, r.URL.Path)

	next(rw, r)

	status := rw.(negroni.ResponseWriter).Status()
	if status > 499 && status < 599 {
		l.Logger.Warnf("upstream: %v %s  %s %s", status, http.StatusText(status), r.Method, r.URL.Path)
	} else {
		l.Logger.Debugf("upstream: %v %s  %s %s", status, http.StatusText(status), r.Method, r.URL.Path)
	}
}
