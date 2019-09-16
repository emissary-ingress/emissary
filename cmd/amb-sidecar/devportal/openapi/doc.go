package openapi

import (
	"net/url"

	"github.com/Jeffail/gabs"
	log "github.com/sirupsen/logrus"
)

type OpenAPIDoc struct {
	JSON *gabs.Container
}

// Create OpenAPI 3.0 JSON spec with URL based on routing information.
// TODO maybe need to support Swagger 2.0.
func NewOpenAPI(jsonDoc []byte, base_url string, prefix string) *OpenAPIDoc {
	logger := log.WithFields(log.Fields{"subsytem": "openapi"})
	logger.WithFields(log.Fields{"base_url": base_url, "prefix": prefix}).Debug(
		"Trying to create new OpenAPI doc")
	// TODO need to support YAML.
	result, err := gabs.ParseJSON(jsonDoc)
	if err != nil {
		logger.WithError(err).Error("Failed to parse OpenAPI JSON")
		return nil
	}

	// Get prefix out of first server URL. E.g. if it's
	// http://example.com/v1, we want to to add /v1 after the Ambassador
	// prefix.
	existingPrefix := ""
	currentServer := result.S("servers").Index(0).S("url").Data()
	log.WithFields(log.Fields{"url": currentServer}).Debug(
		"Checking first server's URL (if any)")
	if currentServer != nil {
		existingUrl, err := url.Parse(currentServer.(string))
		if err == nil {
			existingPrefix = existingUrl.Path
		} else {
			logger.WithFields(
				log.Fields{"error": err, "url": currentServer}).Error(
				"Failed to parse 'servers' URL")
		}
	}
	result.Delete("servers")
	result.Array("servers")
	result.ArrayAppend(0, "servers")
	server, _ := result.S("servers").ObjectI(0)
	serverURL := base_url + existingPrefix + prefix
	logger.WithFields(log.Fields{"url": serverURL}).Debug("Creating OpenAPI doc with public URL")
	server.Set(serverURL, "url")
	return &OpenAPIDoc{JSON: result}
}
