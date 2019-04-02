package openapi

import (
	"github.com/Jeffail/gabs"
)

type OpenAPIDoc struct {
	JSON *gabs.Container
}

// Create OpenAPI 3.0 JSON spec with URL based on routing information:
func NewOpenAPI(json_doc interface{}, base_url string, prefix string) *OpenAPIDoc {
	container, err := gabs.Consume(json_doc)
	if err != nil {
		return nil
	}

	// TODO need to handle case where there's a prefix on the existing
	// server URL, e.g. /v1.

	// Make a copy, so we don't mutate the original:
	result, _ := gabs.ParseJSON(container.EncodeJSON())
	result.Delete("servers")
	result.Array("servers")
	result.ArrayAppend(0, "servers")
	server, _ := result.S("servers").ObjectI(0)
	server.Set(base_url+prefix, "url")
	return &OpenAPIDoc{JSON: result}
}
