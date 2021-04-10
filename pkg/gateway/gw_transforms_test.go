package gateway_test

import (
	"io/ioutil"
	"net/http"
	"testing"

	"github.com/datawire/ambassador/pkg/envoy"
	"github.com/datawire/ambassador/pkg/gateway"
	"github.com/datawire/ambassador/pkg/kates"
	"github.com/stretchr/testify/require"
	"gotest.tools/assert"
)

func TestGatewayMatches(t *testing.T) {
	envoy.SetupRequestLogger(t, ":9000", ":9002")
	e := envoy.SetupEnvoyController(t, ":8003")
	envoy.SetupEnvoy(t, envoy.GetLoopbackAddr(8003), "8080:8080")

	d := makeDispatcher(t)

	// One rule for each type of path match (exact, prefix, regex) and each type of header match
	// (exact and regex).
	err := d.UpsertYaml(`
---
kind: Gateway
apiVersion: networking.x-k8s.io/v1alpha1
metadata:
  name: my-gateway
spec:
  listeners:
  - protocol: HTTP
    port: 8080
---
kind: HTTPRoute
apiVersion: networking.x-k8s.io/v1alpha1
metadata:
  name: my-route
spec:
  rules:
  - matches:
    - path:
        type: Exact
        value: /exact
    forwardTo:
    - serviceName: foo-backend-1
      weight: 100
  - matches:
    - path:
        type: Prefix
        value: /prefix
    forwardTo:
    - serviceName: foo-backend-1
      weight: 100
  - matches:
    - path:
        type: RegularExpression
        value: "/regular_expression(_[aA]+)?"
    forwardTo:
    - serviceName: foo-backend-1
      weight: 100
  - matches:
    - headers:
        type: Exact
        values:
          exact: foo
    forwardTo:
    - serviceName: foo-backend-1
      weight: 100
  - matches:
    - headers:
        type: RegularExpression
        values:
          regular_expression: "foo(_[aA]+)?"
    forwardTo:
    - serviceName: foo-backend-1
      weight: 100
`)

	require.NoError(t, err)
	err = d.Upsert(makeEndpoint("default", "foo-backend-1", envoy.GetLoopbackIp(), 9000))
	require.NoError(t, err)
	err = d.Upsert(makeEndpoint("default", "foo-backend-2", envoy.GetLoopbackIp(), 9001))
	require.NoError(t, err)

	version, snapshot := d.GetSnapshot()
	status := e.Configure("test-id", version, *snapshot)
	if status != nil {
		t.Fatalf("envoy error: %s", status.Message)
	}

	assertGet(t, "http://127.0.0.1:8080/exact", 200, "Hello World")
	assertGet(t, "http://127.0.0.1:8080/exact/foo", 404, "")
	assertGet(t, "http://127.0.0.1:8080/prefix", 200, "Hello World")
	assertGet(t, "http://127.0.0.1:8080/prefix/foo", 200, "Hello World")

	assertGet(t, "http://127.0.0.1:8080/regular_expression", 200, "Hello World")
	assertGet(t, "http://127.0.0.1:8080/regular_expression_a", 200, "Hello World")
	assertGet(t, "http://127.0.0.1:8080/regular_expression_aaaaaaaa", 200, "Hello World")
	assertGet(t, "http://127.0.0.1:8080/regular_expression_aaAaaaAa", 200, "Hello World")
	assertGet(t, "http://127.0.0.1:8080/regular_expression_aaAaaaAab", 404, "")

	assertGetHeader(t, "http://127.0.0.1:8080", "exact", "foo", 200, "Hello World")
	assertGetHeader(t, "http://127.0.0.1:8080", "exact", "bar", 404, "")
	assertGetHeader(t, "http://127.0.0.1:8080", "regular_expression", "foo", 200, "Hello World")
	assertGetHeader(t, "http://127.0.0.1:8080", "regular_expression", "foo_aaaaAaaaa", 200, "Hello World")
	assertGetHeader(t, "http://127.0.0.1:8080", "regular_expression", "foo_aaaaAaaaab", 404, "")
	assertGetHeader(t, "http://127.0.0.1:8080", "regular_expression", "bar", 404, "")
}

func makeDispatcher(t *testing.T) *gateway.Dispatcher {
	d := gateway.NewDispatcher()
	err := d.Register("Gateway", gateway.Compile_Gateway)
	require.NoError(t, err)
	err = d.Register("HTTPRoute", gateway.Compile_HTTPRoute)
	require.NoError(t, err)
	err = d.Register("Endpoints", gateway.Compile_Endpoints)
	require.NoError(t, err)
	return d
}

func makeEndpoint(namespace, name, ip string, port int) *kates.Endpoints {
	ports := []kates.EndpointPort{{Port: int32(port)}}
	addrs := []kates.EndpointAddress{{IP: ip}}

	return &kates.Endpoints{
		TypeMeta:   kates.TypeMeta{Kind: "Endpoints"},
		ObjectMeta: kates.ObjectMeta{Namespace: namespace, Name: name},
		Subsets:    []kates.EndpointSubset{{Addresses: addrs, Ports: ports}},
	}
}

func assertGet(t *testing.T, url string, code int, expected string) {
	resp, err := http.Get(url)
	require.NoError(t, err)
	require.Equal(t, code, resp.StatusCode)
	actual, err := ioutil.ReadAll(resp.Body)
	require.NoError(t, err)
	assert.Equal(t, expected, string(actual))
}

func assertGetHeader(t *testing.T, url, header, value string, code int, expected string) {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	require.NoError(t, err)
	req.Header.Set(header, value)
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	require.Equal(t, code, resp.StatusCode)
	actual, err := ioutil.ReadAll(resp.Body)
	require.NoError(t, err)
	assert.Equal(t, expected, string(actual))
}
