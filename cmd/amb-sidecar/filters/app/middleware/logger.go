package middleware

import (
	"context"
	"fmt"
	"net/http"
	"runtime"
	"time"

	uuid "github.com/satori/go.uuid"

	"github.com/datawire/apro/cmd/amb-sidecar/types"
)

// Logger middleware for logging HTTP calls.
type Logger struct {
	Logger types.Logger
}

func (l *Logger) ServeHTTP(rw http.ResponseWriter, r *http.Request, next http.HandlerFunc) {
	start := time.Now()
	requestID := uuid.NewV4().String()
	sublogger := l.Logger.WithField("REQUEST_ID", requestID)

	sublogger.Infof("[HTTP %s] %s %s", r.Method, r.Host, r.URL.Path)

	// try/catch is spelled awefully funny in Go
	func() {
		defer func() {
			if err := recover(); err != nil {
				// catch
				const stacksize = 64 << 10 // net/http uses 64<<10, negroni.Recovery uses 1024*8 by default
				stack := make([]byte, stacksize)
				stack = stack[:runtime.Stack(stack, false)]
				sublogger.Errorf("[HTTP] panic: %v\n%s", err, stack)

				rw.Header().Set("Content-Type", "text/html; charset=utf-8")
				rw.WriteHeader(http.StatusInternalServerError)
				fmt.Fprintf(rw, `
<html>
  <head>
    <title>HTTP 500: Internal Server Error</title>
  <head>
  <body>
    <h1>HTTP 500: Internal Server Error</h1>
    <p><code>REQUEST_ID=%s</code></p>
  </body>
</html>`, requestID)
			}
		}()
		// try
		next(rw, r.WithContext(context.WithValue(r.Context(), loggerContextKey{}, sublogger)))
	}()

	status := rw.(interface{ Status() int }).Status()
	sublogger.Infof("[HTTP %v] %s %s (%v)", status, r.Method, r.URL.Path, time.Since(start))
}

func GetLogger(r *http.Request) types.Logger {
	return r.Context().Value(loggerContextKey{}).(types.Logger)
}

type loggerContextKey struct{}
