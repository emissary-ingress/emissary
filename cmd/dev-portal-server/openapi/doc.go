package openapi

import (
	"github.com/Jeffail/gabs"
	"log"
)

type OpenAPIDoc struct {
	JSON *gabs.Container
}

// Create OpenAPI 3.0 JSON spec with URL based on routing information.
// TODO maybe need to support Swagger 2.0.
func NewOpenAPI(jsonDoc []byte, base_url string, prefix string) *OpenAPIDoc {
	// TODO need to support YAML.
	result, err := gabs.ParseJSON(jsonDoc)
	if err != nil {
		log.Print(err)
		return nil
	}
	// TODO need to handle case where there's a prefix on the existing
	// server URL, e.g. /v1.
	result.Delete("servers")
	result.Array("servers")
	result.ArrayAppend(0, "servers")
	server, _ := result.S("servers").ObjectI(0)
	server.Set(base_url+prefix, "url")
	return &OpenAPIDoc{JSON: result}
}
