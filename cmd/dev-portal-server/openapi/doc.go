package openapi

import (
	"strings"
	"github.com/Jeffail/gabs"
)

type OpenAPIDoc struct {
	JSON *gabs.Container
}

// Add an extra prefix to all prefixes in the OpenAPI/Swagger document.
func AddPrefix(doc *OpenAPIDoc, prefix string) *OpenAPIDoc {
	prefix = strings.TrimSuffix(prefix, "/")
	// Make a copy, so we don't mutate the original:
	result, _ := gabs.ParseJSON(doc.JSON.EncodeJSON())
	paths, _ := doc.JSON.S("paths").ChildrenMap()
	for key, child := range paths {
		result.S("paths." + key).Delete()
		result.Set(child, "paths", prefix + key)
	}
	return &OpenAPIDoc{JSON: result}
}
