package kates

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidation(t *testing.T) {
	ctx, client := testClient(t, nil)

	version, err := client.ServerVersion()
	require.NoError(t, err)
	if version.Major == "1" && version.Minor == "10" {
		t.Skip("skipping test because kubernetes version is too old")
		return
	} else {
		t.Log("Kubernetes Version", version)
	}

	// create a crd with schema validations so we can test that they work
	objs, err := ParseManifests(CRD)
	require.NoError(t, err)
	require.Equal(t, len(objs), 1)
	crd := objs[0].(*CustomResourceDefinition)

	defer func() {
		assert.NoError(t, client.Delete(ctx, crd, nil))
	}()

	require.NoError(t, client.Create(ctx, crd, crd))

	require.NoError(t, client.WaitFor(ctx, crd.GetName()))

	// check that an invalid crd errors out
	validator, err := NewValidator(client, nil)
	require.NoError(t, err)
	err = validator.Validate(ctx, map[string]interface{}{
		"apiVersion": "test.io/v1",
		"kind":       "TestValidation",
		"spec": map[string]interface{}{
			"ambassador_id": "three", //[]interface{}{"one", "two", 3},
			"foo":           "bar",
			"circuit_breakers": []map[string]interface{}{{
				"priority": "blah",
			}},
		},
		"foo": "bar",
	})
	assert.Error(t, err)
	t.Log(err)

	// check that a valid crd passes
	err = validator.Validate(ctx, map[string]interface{}{
		"apiVersion": "test.io/v1",
		"kind":       "TestValidation",
		"spec": map[string]interface{}{
			"ambassador_id": []interface{}{"one", "two", "three"},
			"circuit_breakers": []map[string]interface{}{{
				"priority": "high",
			}},
		},
	})
	assert.NoError(t, err)

	// check that non-crds validate
	err = validator.Validate(ctx, map[string]interface{}{
		"apiVersion": "v1",
		"kind":       "Service",
	})
	assert.NoError(t, err)
}

var CRD = `
apiVersion: apiextensions.k8s.io/v1
kind: CustomResourceDefinition
metadata:
  name: testvalidations.test.io
spec:
  conversion:
    strategy: None
  group: test.io
  names:
    kind: TestValidation
    plural: testvalidations
    singular: testvalidation
  scope: Namespaced
  versions:
  - name: v1
    schema:
      openAPIV3Schema:
        description: Mapping is the Schema for the mappings API
        properties:
          apiVersion:
            description: 'APIVersion defines the versioned schema of this representation
              of an object. Servers should convert recognized schemas to the latest
              internal value, and may reject unrecognized values. More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#resources'
            type: string
          kind:
            description: 'Kind is a string value representing the REST resource this
              object represents. Servers may infer this from the endpoint the client
              submits requests to. Cannot be updated. In CamelCase. More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#types-kinds'
            type: string
          metadata:
            type: object
          spec:
            properties:
              add_linkerd_headers:
                type: boolean
              add_request_headers:
                items:
                  type: string
                type: array
              add_response_headers:
                items:
                  type: string
                type: array
              ambassador_id:
                description: '"metadata_labels": {     "type": "object",     "additionalProperties":
                  { "type": [ "string", "boolean" ] } },'
                items:
                  type: string
                type: array
              auto_host_rewrite:
                type: boolean
              bypass_auth:
                type: boolean
              case_sensitive:
                type: boolean
              circuit_breakers:
                items:
                  properties:
                    max_connections:
                      format: int32
                      type: integer
                    max_pending_requests:
                      format: int32
                      type: integer
                    max_requests:
                      format: int32
                      type: integer
                    max_retries:
                      format: int32
                      type: integer
                    priority:
                      enum:
                      - default
                      - high
                      type: string
                  type: object
                type: array
              cluster_idle_timeout_ms:
                format: int32
                type: integer
              connect_timeout_ms:
                format: int32
                type: integer
              cors:
                properties:
                  credentials:
                    type: boolean
                  exposed_headers:
                    items:
                      type: string
                    type: array
                  headers:
                    items:
                      type: string
                    type: array
                  max_age:
                    type: string
                  methods:
                    items:
                      type: string
                    type: array
                  origins:
                    items:
                      type: string
                    type: array
                type: object
              enable_ipv4:
                type: boolean
              enable_ipv6:
                type: boolean
              generation:
                format: int32
                type: integer
              grpc:
                type: boolean
              headers:
                additionalProperties:
                  type: string
                type: object
              host:
                type: string
              host_redirect:
                type: boolean
              host_regex:
                type: boolean
              host_rewrite:
                type: string
              idle_timeout_ms:
                format: int32
                type: integer
              labels:
                additionalProperties:
                  type: string
                type: object
              load_balancer:
                properties:
                  cookie:
                    properties:
                      name:
                        type: string
                      path:
                        type: string
                      ttl:
                        type: string
                    type: object
                  header:
                    type: string
                  policy:
                    enum:
                    - round_robin
                    - ring_hash
                    - maglev
                    - least_request
                    type: string
                  source_ip:
                    type: boolean
                type: object
              method:
                type: string
              method_regex:
                type: boolean
              outlier_detection:
                type: string
              path_redirect:
                type: string
              precedence:
                format: int32
                type: integer
              prefix:
                type: string
              prefix_regex:
                type: boolean
              priority:
                type: string
              regex_headers:
                additionalProperties:
                  type: string
                type: object
              remove_request_headers:
                items:
                  type: string
                type: array
              remove_response_headers:
                items:
                  type: string
                type: array
              resolver:
                type: string
              retry_policy:
                properties:
                  num_retries:
                    format: int32
                    type: integer
                  per_try_timeout:
                    type: string
                  retry_on:
                    enum:
                    - 5xx
                    - gateway-error
                    - connect-failure
                    - retriable-4xx
                    - refused-stream
                    - retriable-status-codes
                    type: string
                type: object
              rewrite:
                type: string
              service:
                type: string
              shadow:
                type: boolean
              timeout_ms:
                format: int32
                type: integer
              tls:
                type: string
              use_websocket:
                type: boolean
              weight:
                format: int32
                type: integer
            type: object
          status:
            description: MappingStatus defines the observed state of Mapping
            type: object
        type: object
    served: true
    storage: true
`
