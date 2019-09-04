package internalhandler

import (
	"context"
	"crypto/subtle"
	"net/http"

	"github.com/datawire/apro/lib/filterapi"
)

type InternalFilter struct {
	Secret string
}

func (f *InternalFilter) Filter(ctx context.Context, r *filterapi.FilterRequest) (filterapi.FilterResponse, error) {
	secret := r.GetRequest().GetHttp().GetHeaders()["X-Ambassador-Internal-Auth"]
	if subtle.ConstantTimeCompare([]byte(f.Secret), []byte(secret)) != 0 {
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
