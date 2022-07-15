package agent

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

const (
	validOpenAPI2Contract = `{
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
    "servers": [
        {
            "url": "https://23.251.148.46/backend"
        }
    ],
    "swagger": "2.0",
    "tags": [
        {
            "description": "The Account API",
            "name": "account"
        }
    ]
}`
)

func TestParseToOpenAPIV3ConvertToV2(t *testing.T) {
	// when
	v3Doc, err := parseToOpenAPIV3([]byte(validOpenAPI2Contract))

	// then
	assert.NoError(t, err)
	assert.NotNil(t, v3Doc)
}
