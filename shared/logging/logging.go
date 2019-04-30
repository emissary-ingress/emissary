package logging

import (
	"net/http"

	log "github.com/sirupsen/logrus"
)

type wrappedResponseWriter struct {
	http.ResponseWriter
	statusCode int
}

func newLoggingResponseWriter(w http.ResponseWriter) *wrappedResponseWriter {
	return &wrappedResponseWriter{w, http.StatusOK}
}

func (lw *wrappedResponseWriter) WriteHeader(code int) {
	lw.statusCode = code
	lw.ResponseWriter.WriteHeader(code)
}

// A middleware that can be used with gorilla to log all HTTP requests and their
// response code, by doing router.Use(logging.LoggingMiddleware).
func LoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		loggingWriter := newLoggingResponseWriter(w)
		next.ServeHTTP(loggingWriter, r)
		log.WithFields(log.Fields{
			"subsystem":   "server",
			"request":     r.RequestURI,
			"method":      r.Method,
			"status_code": loggingWriter.statusCode,
		}).Info("HTTP request")
	})
}
