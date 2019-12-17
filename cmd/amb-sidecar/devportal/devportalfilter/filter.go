package devportalfilter

import (
	"context"
	"net/http"

	"github.com/datawire/apro/cmd/amb-sidecar/internalaccess"
	"github.com/datawire/apro/lib/filterapi"
	"github.com/datawire/apro/lib/filterapi/filterutil"
)

type DevPortalFilter struct {
	secret *internalaccess.InternalSecret
}

func MakeDevPortalFilter() *DevPortalFilter {
	return &DevPortalFilter{
		secret: internalaccess.GetInternalSecret(),
	}
}

func (f *DevPortalFilter) Filter(ctx context.Context, r *filterapi.FilterRequest) (filterapi.FilterResponse, error) {
	secret := filterutil.GetHeader(r).Get("X-Ambassador-DevPortal-Auth")
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
