package openapi

import (
	"github.com/Jeffail/gabs"
	"strings"
)

type OpenAPIDoc struct {
	JSON *gabs.Container
}

// Add an extra prefix to all prefixes in the OpenAPI/Swagger document.
func AddPrefix(json_doc interface{}, prefix string) *OpenAPIDoc {
	prefix = strings.TrimSuffix(prefix, "/")
	// Make a copy, so we don't mutate the original:
	container, err := gabs.Consume(json_doc)
	if err != nil {
		return nil
	}
	result, _ := gabs.ParseJSON(container.EncodeJSON())
	paths, _ := container.S("paths").ChildrenMap()
	for key, child := range paths {
		result.S("paths." + key).Delete()
		result.Set(child, "paths", prefix+key)
	}
	return &OpenAPIDoc{JSON: result}
}
