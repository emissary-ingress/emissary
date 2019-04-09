package server

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Jeffail/gabs"
	. "github.com/datawire/apro/cmd/dev-portal-server/kubernetes"
	. "github.com/datawire/apro/cmd/dev-portal-server/openapi"
	. "github.com/onsi/gomega"
)

// We can add a service to the internal memory representation, and it gets
// exposed via a HTTP API.
func TestAddThenGetViaHTTP(t *testing.T) {
	openapiJSON := []byte(` {
        "openapi": "3.0.0",
        "info": {
            "title": "My API",
            "description": "description",
            "version": 1.0
        },
        "servers": [
            {
                "url": "http://api.example.com/"
            }
        ],
        "paths": {
            "/widgets": {
                "get": {
                    "summary": "Get widgets.",
                    "responses": {
                        "200": {
                            "description": "A JSON array of widgets",
                            "content": {
                                "application/json": {
                                    "schema": {
                                        "type": "array",
                                        "items": {
                                            "type": "string"
                                        }
                                    }
                                }
                            }
                        }
                    }
                }
            }
        }
    }
`)

	g := NewGomegaWithT(t)
	s := NewServer()
	baseURL := "https://example.com"
	prefix := "/foo"
	svc := Service{Name: "mysvc", Namespace: "myns"}

	// We add a service:
	s.getServiceAdd()(svc, baseURL, prefix, openapiJSON)

	// We can retrieve the updated OpenAPI via HTTP:
	req, err := http.NewRequest(
		"GET", "/openapi/services/myns/mysvc/openapi.json", nil)
	if err != nil {
		t.Fatal(err)
	}
	rr := httptest.NewRecorder()
	expectedDoc := NewOpenAPI(openapiJSON, baseURL, prefix).JSON
	s.router.ServeHTTP(rr, req)
	g.Expect(rr.Code).To(Equal(http.StatusOK))
	resultJson, _ := gabs.ParseJSON(rr.Body.Bytes())
	g.Expect(resultJson).To(Equal(expectedDoc))
}
