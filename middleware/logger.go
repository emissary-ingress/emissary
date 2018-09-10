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
	next(rw, r)
	status := rw.(negroni.ResponseWriter).Status()
	if status > 499 && status < 599 {
		l.Logger.Errorf("%v %s  %s %s", status, http.StatusText(status), r.Method, r.URL.Path)
	} else {
		l.Logger.Infof("%v %s  %s %s", status, http.StatusText(status), r.Method, r.URL.Path)
	}
}
