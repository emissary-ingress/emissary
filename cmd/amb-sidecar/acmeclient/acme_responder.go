// responder.go deals with responding when the ACME server tries
// talking to us.

package acmeclient

import (
	"net/http"
	"strings"

	"github.com/mediocregopher/radix.v2/pool"
	"github.com/pkg/errors"

	"github.com/datawire/apro/cmd/amb-sidecar/filters/handler/middleware"
)

func NewChallengeHandler(redisPool *pool.Pool) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// sanity check
		if !strings.HasPrefix(r.URL.Path, "/.well-known/acme-challenge/") {
			panic("a programmer installed the ACME challenge handler wrong")
		}

		redisClient, err := redisPool.Get()
		if err != nil {
			middleware.ServeErrorResponse(w, r.Context(), http.StatusBadGateway, err, nil)
			return
		}
		defer redisPool.Put(redisClient)

		challengeToken := strings.TrimPrefix(r.URL.Path, "/.well-known/acme-challenge/")
		challengeResponse, err := redisClient.Cmd("GET", "acme-challenge:"+challengeToken).Bytes()
		if err != nil {
			http.NotFound(w, r)
			return
		}

		if r.Method != "GET" {
			middleware.ServeErrorResponse(w, r.Context(), http.StatusMethodNotAllowed,
				errors.New("method not allowed"), nil)
			return
		}

		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.Write(challengeResponse)
		r.Close = true
	})
}
