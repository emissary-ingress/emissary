#!/hint/sh

# "Releasable" images
AES_IMAGE=$(                         sed -n 2p docker/model-cluster-aes-plugins.docker.push.cluster) # XXX: not releasable because plugins
CONSUL_CONNECT_INTEGRATION_IMAGE=$(  sed -n 2p docker/consul_connect_integration.docker.push.cluster)
PROXY_IMAGE=$(                       sed -n 2p docker/traffic-proxy.docker.push.cluster)
SIDECAR_IMAGE=$(                     sed -n 2p docker/app-sidecar.docker.push.cluster)

# Model cluster / example images
MODEL_CLUSTER_APP_IMAGE=$(           sed -n 2p docker/model-cluster-app.docker.push.cluster)
MODEL_CLUSTER_GRPC_AUTH_IMAGE=$(     sed -n 2p docker/model-cluster-grpc-auth.docker.push.cluster)
MODEL_CLUSTER_HTTP_AUTH_IMAGE=$(     sed -n 2p docker/model-cluster-http-auth.docker.push.cluster)
MODEL_CLUSTER_LOAD_GRPC_AUTH_IMAGE=$(sed -n 2p docker/model-cluster-load-grpc-auth.docker.push.cluster)
MODEL_CLUSTER_LOAD_HTTP_AUTH_IMAGE=$(sed -n 2p docker/model-cluster-load-http-auth.docker.push.cluster)
MODEL_CLUSTER_LOGOUT_IMAGE=$(        sed -n 2p docker/model-cluster-logout.docker.push.cluster)
MODEL_CLUSTER_OPENAPI_SERVICE=$(     sed -n 2p docker/model-cluster-openapi-service.docker.push.cluster)
MODEL_CLUSTER_UAA_IMAGE=$(           sed -n 2p docker/model-cluster-uaa.docker.push.cluster)

# Loadtest images
LOADTEST_GENERATOR_IMAGE=$(          sed -n 2p docker/loadtest-generator.docker.push.cluster)

# 03-ambassador-pro-*.yaml
AMBASSADOR_LICENSE_KEY_V0=eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJpZCI6ImRldiIsImV4cCI6NDcwMDgyNjEzM30.wCxi5ICR6C5iEz6WkKpurNItK3zER12VNhM8F1zGkA8
# Created with `./bin/apictl-key create --id=dev --expiration=$((100*365)) --features=filter,ratelimit,traffic,devportal,certified-envoy`
AMBASSADOR_LICENSE_KEY_V1=eyJhbGciOiJQUzUxMiIsInR5cCI6IkpXVCJ9.eyJsaWNlbnNlX2tleV92ZXJzaW9uIjoidjEiLCJjdXN0b21lcl9pZCI6ImRldiIsImVuYWJsZWRfZmVhdHVyZXMiOlsiZmlsdGVyIiwicmF0ZWxpbWl0IiwidHJhZmZpYyIsImRldnBvcnRhbCIsImNlcnRpZmllZC1lbnZveSJdLCJleHAiOjQ3MjAxMTM4NTksImlhdCI6MTU2NjUxMzg1OSwibmJmIjoxNTY2NTEzODU5fQ.ZPj034sI-yYlQemj9U9u6OzPKx4vrBf0Xv_NlvPSWhvzIlvTkJ-eDUxeWcMEgIjxZe6R2D-B6uRAtJLqFEFu2hA6DATzKFhk_4OTitpAwgVYWkHPy3Cd2rOhTx_vqcT3kYQei3OkBIIPkNvU-nbvAfL3CVICC083yW5sdckcmclsFY_fTOvaGi95bEeQAVh7e90b64yYz9P8zLbqwQ9l-rMvkSoh5euLsdRRT2g98ff7rPZIOdeiO4JQ9IbwukO21Z2Nzo7EOdgUesI6DBfvw7i2KisRSIaO-lVwnDYsrqPhfjFmzG3tPlHfy3qn16JPZ1RDxziRyJ8ZgSrtmEBpBA
# Created with `bin_darwin_amd64/apictl-key create --id dev --expiration 36500 --features filter,ratelimit,traffic,devportal,certified-envoy,local-devportal`
AMBASSADOR_LICENSE_KEY_V2=eyJhbGciOiJQUzUxMiIsInR5cCI6IkpXVCJ9.eyJsaWNlbnNlX2tleV92ZXJzaW9uIjoidjIiLCJjdXN0b21lcl9pZCI6ImRldiIsImVuYWJsZWRfZmVhdHVyZXMiOlsiZmlsdGVyIiwicmF0ZWxpbWl0IiwidHJhZmZpYyIsImRldnBvcnRhbCIsImNlcnRpZmllZC1lbnZveSIsImxvY2FsLWRldnBvcnRhbCJdLCJlbmZvcmNlZF9saW1pdHMiOltdLCJleHAiOjQ3MjQzOTUxMzksImlhdCI6MTU3MDc5NTEzOSwibmJmIjoxNTcwNzk1MTM5fQ.gfMLm9EgmXtyZc-W3LpfopVSEyTwoxOviNGTGsqbAmHAI0Kf7b9drdM1LCO64BKi5vS3zbJK64jxkw6eH8jHstmO4PCRZ--Vz4CbTw8k7zzm6vPC-YHgjWGeIG7ovPGCdnzxohYEmqmuEkjomFS7_FXRe38AYNuaILMVrUg1uoUfcJt3k6tMJ2j5KBsYFemlt6EQCG1SpnO_lygXm6wvaNHBPHa3nHdF5GTbTmz1KIMgYvGflwDEmbuqDwo8iQy1Hqck2kaJl2C3E_63BcAlwimN9eMZikrzXS6FC0vaEo-TVnG9wX4QqKogKumCi9CGzUpVlYvo7ay79-40gqOcBA
AMBASSADOR_LICENSE_KEY=$AMBASSADOR_LICENSE_KEY_V2
