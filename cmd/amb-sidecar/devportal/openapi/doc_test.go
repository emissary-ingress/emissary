package openapi

import (
	"testing"

	"github.com/Jeffail/gabs"
	. "github.com/onsi/gomega"
)

// The servers key of the OpenAPI doc is updated with a new URL. This is the
// case where there is no URL prefix on existing server URLs.
func TestUpdateServersNoExistingPrefix(t *testing.T) {
	g := NewGomegaWithT(t)

	// At some point we might want to validate OpenAPI versions, and that
	// it's a valid document, in which case this will need to be a real
	// OpenAPI document.
	expected_json, _ := gabs.ParseJSON(
		[]byte(`{"untouched": "X","servers": [{"url": "http://mybase/myroute/is"}]}`))
	doc := NewOpenAPI(
		[]byte(`{"untouched": "X","servers": []}`), "http://mybase",
		"/myroute/is")
	g.Expect(doc).To(Equal(&OpenAPIDoc{JSON: expected_json}))
}

// The servers key of the OpenAPI doc is updated with a new URL, which includes the existing path prefix on the original URL.
func TestUpdateServersWithExistingPrefix(t *testing.T) {
	g := NewGomegaWithT(t)

	// At some point we might want to validate OpenAPI versions, and that
	// it's a valid document, in which case this will need to be a real
	// OpenAPI document.
	expected_json, _ := gabs.ParseJSON(
		[]byte(`{"untouched": "X","servers": [{"url": "http://mybase/v1/myroute/is"}]}`))
	doc := NewOpenAPI(
		[]byte(`{"untouched": "X","servers": [{"url": "http://whatever/v1"}]}`), "http://mybase",
		"/myroute/is")
	g.Expect(doc).To(Equal(&OpenAPIDoc{JSON: expected_json}))
}
