#!/hint/sh

AES_EXAMPLE_PLUGINS_IMAGE=$(          sed -n 2p docker/aes-example-plugins.docker.push.dev)
CONSUL_CONNECT_INTEGRATION_IMAGE=$(   sed -n 2p docker/consul_connect_integration.docker.push.dev)
MODEL_CLUSTER_APP_IMAGE=$(            sed -n 2p docker/model-cluster-app.docker.push.dev)
MODEL_CLUSTER_GRPC_AUTH_IMAGE=$(      sed -n 2p docker/model-cluster-grpc-auth.docker.push.dev)
MODEL_CLUSTER_HTTP_AUTH_IMAGE=$(      sed -n 2p docker/model-cluster-http-auth.docker.push.dev)
MODEL_CLUSTER_LOGOUT_IMAGE=$(         sed -n 2p docker/model-cluster-logout.docker.push.dev)
MODEL_CLUSTER_OPENAPI_SERVICE_IMAGE=$(sed -n 2p docker/model-cluster-openapi-service.docker.push.dev)
MODEL_CLUSTER_UAA_IMAGE=$(            sed -n 2p docker/model-cluster-uaa.docker.push.dev)

#AES_IMAGE=$(                          sed -n 2p docker/model-cluster-aes-plugins.docker.push.dev) # XXX: not releasable because plugins
#MODEL_CLUSTER_LOAD_GRPC_AUTH_IMAGE=$( sed -n 2p docker/model-cluster-load-grpc-auth.docker.push.dev)
#MODEL_CLUSTER_LOAD_HTTP_AUTH_IMAGE=$( sed -n 2p docker/model-cluster-load-http-auth.docker.push.dev)
#LOADTEST_GENERATOR_IMAGE=$(           sed -n 2p docker/loadtest-generator.docker.push.dev)

# Created with `bin_darwin_amd64/apictl-key create --id dev --expiration 36500 --features filter,ratelimit,traffic,devportal,certified-envoy,local-devportal`
AMBASSADOR_LICENSE_KEY=eyJhbGciOiJQUzUxMiIsInR5cCI6IkpXVCJ9.eyJsaWNlbnNlX2tleV92ZXJzaW9uIjoidjEiLCJjdXN0b21lcl9pZCI6ImRldiIsImVuYWJsZWRfZmVhdHVyZXMiOlsiZmlsdGVyIiwicmF0ZWxpbWl0IiwidHJhZmZpYyIsImRldnBvcnRhbCIsImNlcnRpZmllZC1lbnZveSJdLCJleHAiOjQ3MjAxMTM4NTksImlhdCI6MTU2NjUxMzg1OSwibmJmIjoxNTY2NTEzODU5fQ.ZPj034sI-yYlQemj9U9u6OzPKx4vrBf0Xv_NlvPSWhvzIlvTkJ-eDUxeWcMEgIjxZe6R2D-B6uRAtJLqFEFu2hA6DATzKFhk_4OTitpAwgVYWkHPy3Cd2rOhTx_vqcT3kYQei3OkBIIPkNvU-nbvAfL3CVICC083yW5sdckcmclsFY_fTOvaGi95bEeQAVh7e90b64yYz9P8zLbqwQ9l-rMvkSoh5euLsdRRT2g98ff7rPZIOdeiO4JQ9IbwukO21Z2Nzo7EOdgUesI6DBfvw7i2KisRSIaO-lVwnDYsrqPhfjFmzG3tPlHfy3qn16JPZ1RDxziRyJ8ZgSrtmEBpBA
