package openapi

import (
	"log"
	"net/url"

	"github.com/Jeffail/gabs"
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

	// Get prefix out of first server URL. E.g. if it's
	// http://example.com/v1, we want to to add /v1 after the Ambassador
	// prefix.
	existingPrefix := ""
	currentServer := result.S("servers").Index(0).S("url").Data()
	if currentServer != nil {
		existingUrl, err := url.Parse(currentServer.(string))
		if err == nil {
			existingPrefix = existingUrl.Path
		}
	}
	result.Delete("servers")
	result.Array("servers")
	result.ArrayAppend(0, "servers")
	server, _ := result.S("servers").ObjectI(0)
	server.Set(base_url+existingPrefix+prefix, "url")
	return &OpenAPIDoc{JSON: result}
}
