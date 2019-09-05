package internalhandler

import (
	"context"
	"net/http"

	"github.com/datawire/apro/cmd/amb-sidecar/internal-access/secret"
	"github.com/datawire/apro/lib/filterapi"
)

type InternalFilter struct {
	secret *secret.InternalSecret
}

func MakeInternalFilter() *InternalFilter {
	return &InternalFilter{
		secret: secret.GetInternalSecret(),
	}
}

func (f *InternalFilter) Filter(ctx context.Context, r *filterapi.FilterRequest) (filterapi.FilterResponse, error) {
	secret := r.GetRequest().GetHttp().GetHeaders()["X-Ambassador-Internal-Auth"]
	if f.secret.Compare(secret) != 1 {
		// hide the internal URL from the outside world
		return &filterapi.HTTPResponse{
			StatusCode: http.StatusNotFound,
			Header: http.Header{
				"Content-Type": {"text/plain"},
			},
			Body: "not found",
		}, nil
	}
	return &filterapi.HTTPRequestModification{}, nil
}
