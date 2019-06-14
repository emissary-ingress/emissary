package filterutil

import (
	"net/http"

	"github.com/pkg/errors"

	"github.com/datawire/apro/lib/filterapi"
)

// ApplyRequestModification mutates the filterapi representation of an
// HTTP request according to an HTTPRequestModification response.
func ApplyRequestModification(req *filterapi.FilterRequest, mod *filterapi.HTTPRequestModification) {
	for _, hmod := range mod.Header {
		switch hmod := hmod.(type) {
		case *filterapi.HTTPHeaderAppendValue:
			if cur, ok := req.Request.Http.Headers[http.CanonicalHeaderKey(hmod.Key)]; ok {
				req.Request.Http.Headers[http.CanonicalHeaderKey(hmod.Key)] = cur + "," + hmod.Value
			} else {
				req.Request.Http.Headers[http.CanonicalHeaderKey(hmod.Key)] = hmod.Value
			}
		case *filterapi.HTTPHeaderReplaceValue:
			req.Request.Http.Headers[http.CanonicalHeaderKey(hmod.Key)] = hmod.Value
		default:
			panic(errors.Errorf("unexpected header modification type %T", hmod))
		}
	}
}
