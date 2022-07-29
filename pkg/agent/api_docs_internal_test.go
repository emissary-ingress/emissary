package agent

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

const (
	openAPI3ContractWithBaseURL = `{
	"info": {
		"description": "Quote Service API",
		"title": "Quote Service API",
		"version": "0.1.0"
	},
	"openapi": "3.0.0",
	"paths": {
		"/": {
			"get": {
				"responses": {
					"200": {
						"content": {
							"application/json": {
								"schema": {
									"properties": {
										"quote": {
											"type": "string"
										},
										"server": {
											"type": "string"
										},
										"time": {
											"type": "string"
										}
									},
									"type": "object"
								}
							}
						},
						"description": "A JSON object with a quote and some additional metadata."
					}
				},
				"summary": "Return a randomly selected quote."
			}
		}
	},
	"servers": [{
		"url": "https://app.acme.com/cloud/api/sso/"
	}]
}`
	invalidOpenAPI3Contract = `{
	"info123": {
		"description": "Quote Service API",
		"title": "Quote Service API",
		"version": "0.1.0"
	},
	"openapi": "3.0.0",
	"pathinvalid": {
		"/": {
			"get": {
				"responses": {
					"200": {
						"content": {
							"application/json": {
								"schema": {
									"properties": {
										"quote": {
											"type": "string"
										},
										"server": {
											"type": "string"
										},
										"time": {
											"type": "string"
										}
									},
									"type": "object"
								}
							}
						},
						"description": "A JSON object with a quote and some additional metadata."
					}
				},
				"summary": "Return a randomly selected quote."
			}
		},
		"/debug/": {
			"get": {
				"responses": {
					"200": {
						"content": {
							"application/json": {
								"schema": {
									"properties": {
										"headers": {
											"type": "object"
										},
										"host": {
											"type": "string"
										},
										"proto": {
											"type": "string"
										},
										"remoteaddr": {
											"type": "string"
										},
										"server": {
											"type": "string"
										},
										"time": {
											"type": "string"
										},
										"url": {
											"type": "object"
										}
									},
									"type": "object"
								}
							}
						},
						"description": "A JSON object with debug information about the request and additional metadata."
					}
				},
				"summary": "Return debug information about the request."
			}
		}
	},
	"servers": [{
		"url": "https://23.251.148.46/backend"
	}]
}`
	validOpenAPI3Contract = `{
	"info": {
		"description": "Quote Service API",
		"title": "Quote Service API",
		"version": "0.1.0"
	},
	"openapi": "3.0.0",
	"paths": {
		"/": {
			"get": {
				"responses": {
					"200": {
						"content": {
							"application/json": {
								"schema": {
									"properties": {
										"quote": {
											"type": "string"
										},
										"server": {
											"type": "string"
										},
										"time": {
											"type": "string"
										}
									},
									"type": "object"
								}
							}
						},
						"description": "A JSON object with a quote and some additional metadata."
					}
				},
				"summary": "Return a randomly selected quote."
			}
		},
		"/debug/": {
			"get": {
				"responses": {
					"200": {
						"content": {
							"application/json": {
								"schema": {
									"properties": {
										"headers": {
											"type": "object"
										},
										"host": {
											"type": "string"
										},
										"proto": {
											"type": "string"
										},
										"remoteaddr": {
											"type": "string"
										},
										"server": {
											"type": "string"
										},
										"time": {
											"type": "string"
										},
										"url": {
											"type": "object"
										}
									},
									"type": "object"
								}
							}
						},
						"description": "A JSON object with debug information about the request and additional metadata."
					}
				},
				"summary": "Return debug information about the request."
			}
		}
	},
	"servers": [{
		"url": "https://23.251.148.46/backend"
	}]
}`
	invalidOpenAPI2Contract = `{
	"basePath": "/api",
	"definitionTypo": {
		"Account": {
			"properties": {
				"avatarUrl": {
					"description": "Avatar url",
					"type": "string"
				},
				"id": {
					"description": "Account id",
					"type": "string"
				},
				"identityProvider": {
					"description": "Identity Provider",
					"type": "string"
				},
				"invitationPending": {
					"description": "The user has a pending invitation for this account",
					"type": "boolean"
				},
				"name": {
					"description": "Account name",
					"type": "string"
				}
			},
			"required": [
				"id",
				"name"
			],
			"type": "object"
		},
		"Error": {
			"properties": {
				"errorCode": {
					"enum": [
						"DEMO_CLUSTER_DEPLETED",
						"SUBSCRIPTION_LIMIT_REACHED"
					],
					"type": "string",
					"x-nullable": true
				},
				"message": {
					"type": "string"
				}
			},
			"required": [
				"message"
			],
			"type": "object"
		}
	},
	"paths": {
		"/accounts": {
			"get": {
				"operationId": "listAccounts",
				"produces": [
					"application/json"
				],
				"responses": {
					"200": {
						"description": "successful operation",
						"schema": {
							"items": {
								"$ref": "#/definitions/Account"
							},
							"type": "array"
						}
					},
					"500": {
						"$ref": "#/responses/GenericError"
					}
				},
				"summary": "List the accounts the current user is a member of",
				"tags": [
					"account"
				]
			}
		}
	},
	"responses": {
		"GenericError": {
			"description": "An unexpected occurred on the server",
			"schema": {
				"$ref": "#/definitions/Error"
			}
		}
	},
	"schemes": [
		"http"
	],
	"servers": [{
		"url": "https://23.251.148.46/backend"
	}],
	"swagger": "2.0",
	"tags": [{
		"description": "The Account API",
		"name": "account"
	}]
}`
	validOpenAPI2Contract = `{
	"info": {
		"description": "This API provides the basic functionality for System-A UI operations.",
		"version": "1.0.0",
		"title": "System-A API"
	},
	"basePath": "/api",
	"definitions": {
		"Account": {
			"properties": {
				"avatarUrl": {
					"description": "Avatar url",
					"type": "string"
				},
				"id": {
					"description": "Account id",
					"type": "string"
				},
				"identityProvider": {
					"description": "Identity Provider",
					"type": "string"
				},
				"invitationPending": {
					"description": "The user has a pending invitation for this account",
					"type": "boolean"
				},
				"name": {
					"description": "Account name",
					"type": "string"
				}
			},
			"required": [
				"id",
				"name"
			],
			"type": "object"
		},
		"Error": {
			"properties": {
				"errorCode": {
					"enum": [
						"DEMO_CLUSTER_DEPLETED",
						"SUBSCRIPTION_LIMIT_REACHED"
					],
					"type": "string",
					"x-nullable": true
				},
				"message": {
					"type": "string"
				}
			},
			"required": [
				"message"
			],
			"type": "object"
		}
	},
	"paths": {
		"/accounts": {
			"get": {
				"operationId": "listAccounts",
				"produces": [
					"application/json"
				],
				"responses": {
					"200": {
						"description": "successful operation",
						"schema": {
							"items": {
								"$ref": "#/definitions/Account"
							},
							"type": "array"
						}
					},
					"500": {
						"$ref": "#/responses/GenericError"
					}
				},
				"summary": "List the accounts the current user is a member of",
				"tags": [
					"account"
				]
			}
		}
	},
	"responses": {
		"GenericError": {
			"description": "An unexpected occurred on the server",
			"schema": {
				"$ref": "#/definitions/Error"
			}
		}
	},
	"schemes": [
		"http"
	],
	"servers": [{
		"url": "https://23.251.148.46/backend"
	}],
	"swagger": "2.0",
	"tags": [{
		"description": "The Account API",
		"name": "account"
	}]
}`
)

func TestParseToOpenAPIV3(t *testing.T) {
	t.Run("convert open api v2 to open api v3", func(t *testing.T) {
		// when
		v3Doc, err := parseToOpenAPIV3([]byte(validOpenAPI2Contract))

		// then
		assert.NoError(t, err)
		assert.NotNil(t, v3Doc)
	})
	t.Run("parse open api v3", func(t *testing.T) {
		// when
		v3Doc, err := parseToOpenAPIV3([]byte(validOpenAPI3Contract))

		// then
		assert.NoError(t, err)
		assert.NotNil(t, v3Doc)
	})
	t.Run("parse invalid open api v3", func(t *testing.T) {
		// when
		v3Doc, err := parseToOpenAPIV3([]byte(invalidOpenAPI3Contract))

		// then
		assert.Error(t, err)
		assert.Nil(t, v3Doc)
	})
	t.Run("parse invalid open api v2", func(t *testing.T) {
		// when
		v3Doc, err := parseToOpenAPIV3([]byte(invalidOpenAPI2Contract))

		// then
		assert.Error(t, err)
		assert.Nil(t, v3Doc)
	})
}

func TestNewOpenAPI(t *testing.T) {
	t.Run("convert open api v2 to open api v3", func(t *testing.T) {
		// when
		openAPI := newOpenAPI(
			context.Background(), []byte(openAPI3ContractWithBaseURL),
			"my-api-gateway.com", "/cloud/api/sso/", "/api/sso", true,
		)

		// then
		doc, err := parseToOpenAPIV3(openAPI.JSON)
		if err != nil {
			t.Fatal(err)
		}
		assert.Equal(t, "test", doc.Servers[0].URL)
		assert.NotNil(t, openAPI)
	})
}
