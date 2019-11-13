package server

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Jeffail/gabs"
	. "github.com/onsi/gomega"

	"github.com/datawire/apro/cmd/amb-sidecar/devportal/openapi"
	"github.com/datawire/apro/cmd/amb-sidecar/limiter/mocks"
	"github.com/datawire/apro/lib/licensekeys"
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
	l := mocks.NewMockLimiter()
	s := NewServer("", nil, l)
	baseURL := "https://example.com"
	prefix := "/foo"
	svc := Service{Name: "mysvc", Namespace: "myns"}

	// We add a service:
	err := s.AddService(svc, baseURL, prefix, openapiJSON)
	if err != nil {
		t.Fatal(err)
	}

	// We can retrieve the updated OpenAPI via HTTP:
	req, err := http.NewRequest(
		"GET", "/openapi/services/myns/mysvc/openapi.json", nil)
	if err != nil {
		t.Fatal(err)
	}
	rr := httptest.NewRecorder()
	expectedDoc := openapi.NewOpenAPI(openapiJSON, baseURL, prefix).JSON
	s.router.ServeHTTP(rr, req)
	g.Expect(rr.Code).To(Equal(http.StatusOK))
	resultJson, _ := gabs.ParseJSON(rr.Body.Bytes())
	g.Expect(resultJson).To(Equal(expectedDoc))
}

func TestRedactedDocument(t *testing.T) {
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
	l := mocks.NewMockLimiterWithCounts(map[licensekeys.Limit]int{
		licensekeys.LimitDevPortalServices: 0,
	}, false)
	s := NewServer("", nil, l)
	baseURL := "https://example.com"
	prefix := "/foo"
	svc := Service{Name: "mysvc", Namespace: "myns"}

	err := s.AddService(svc, baseURL, prefix, openapiJSON)
	if err != nil {
		t.Fatal(err)
	}

	req, err := http.NewRequest(
		"GET", "/openapi/services/myns/mysvc/openapi.json", nil)
	if err != nil {
		t.Fatal(err)
	}
	rr := httptest.NewRecorder()
	s.router.ServeHTTP(rr, req)
	g.Expect(rr.Code).To(Equal(http.StatusOK))
	resultJson, _ := gabs.ParseJSON(rr.Body.Bytes())
	g.Expect(resultJson.Path("info.description").Data().(string)).To(Equal("\n# This document exceeds Developer Portal service limit in the license\n\nJust the skeleton of the service URL space will be shown.\n\nPlease contact sales@datawire.io\n\ndescription"))
}

func TestHardLimitDocument(t *testing.T) {
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
	l := mocks.NewMockLimiterWithCounts(map[licensekeys.Limit]int{
		licensekeys.LimitDevPortalServices: 0,
	}, true)
	s := NewServer("", nil, l)
	baseURL := "https://example.com"
	prefix := "/foo"
	svc := Service{Name: "mysvc", Namespace: "myns"}

	err := s.AddService(svc, baseURL, prefix, openapiJSON)
	if err == nil {
		t.Fatal("AddService should return an error on hard limits!")
	}

	req, _ := http.NewRequest(
		"GET", "/openapi/services/myns/mysvc/openapi.json", nil)
	rr := httptest.NewRecorder()
	s.router.ServeHTTP(rr, req)
	g.Expect(rr.Code).To(Equal(http.StatusNotFound))
}

// An unknown OpenAPI doc results in a 404.
func TestOpenAPIDocNotFound(t *testing.T) {
	g := NewGomegaWithT(t)
	l := mocks.NewMockLimiter()
	s := NewServer("", nil, l)
	req, _ := http.NewRequest(
		"GET", "/openapi/services/myns/mysvc/openapi.json", nil)
	rr := httptest.NewRecorder()
	s.router.ServeHTTP(rr, req)
	g.Expect(rr.Code).To(Equal(http.StatusNotFound))
}
