// responder.go deals with responding when the ACME server tries
// talking to us.

package acmeclient

import (
	"context"
	"net/http"
	"strings"

	"github.com/mediocregopher/radix.v2/pool"
	"github.com/pkg/errors"

	"github.com/datawire/apro/cmd/amb-sidecar/filters/handler/middleware"
	"github.com/datawire/apro/lib/filterapi"
)

type challengeHandler struct {
	redisPool *pool.Pool
}

func (h *challengeHandler) Filter(ctx context.Context, request *filterapi.FilterRequest) (filterapi.FilterResponse, error) {
	// sanity check
	urlPath := request.GetRequest().GetHttp().GetPath()
	if !strings.HasPrefix(urlPath, "/.well-known/acme-challenge/") {
		panic("a programmer installed the ACME challenge handler wrong")
	}

	redisClient, err := h.redisPool.Get()
	if err != nil {
		return middleware.NewErrorResponse(ctx, http.StatusBadGateway, err, nil), nil
	}
	defer h.redisPool.Put(redisClient)

	challengeToken := strings.TrimPrefix(urlPath, "/.well-known/acme-challenge/")
	challengeResponse, err := redisClient.Cmd("GET", "acme-challenge:"+challengeToken).Str()
	if err != nil {
		// Fall through to a mapping -- probably 404, but
		// could be a user-provided ACME client.
		return &filterapi.HTTPRequestModification{}, nil
	}

	if request.GetRequest().GetHttp().GetMethod() != http.MethodGet {
		return middleware.NewErrorResponse(ctx, http.StatusMethodNotAllowed,
			errors.New("method not allowed"), nil), nil
	}

	return &filterapi.HTTPResponse{
		StatusCode: http.StatusOK,
		Header: http.Header{
			"Content-Type": {"text/plain; charset=utf-8"},
		},
		Body: challengeResponse,
	}, nil
}

func NewChallengeHandler(redisPool *pool.Pool) filterapi.Filter {
	return &challengeHandler{
		redisPool: redisPool,
	}
}
