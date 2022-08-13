package gateway_test

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	gw "sigs.k8s.io/gateway-api/apis/v1alpha1"

	"github.com/datawire/dlib/dgroup"
	"github.com/datawire/dlib/dlog"
	"github.com/emissary-ingress/emissary/v3/pkg/envoytest"
	"github.com/emissary-ingress/emissary/v3/pkg/gateway"
	"github.com/emissary-ingress/emissary/v3/pkg/kates"
)

func TestGatewayMatches(t *testing.T) {
	t.Parallel()

	ctx := dlog.NewTestContext(t, false)
	grp := dgroup.NewGroup(ctx, dgroup.GroupConfig{
		EnableWithSoftness: true,
		ShutdownOnNonError: true,
	})

	grp.Go("upstream", func(ctx context.Context) error {
		var reqLogger envoytest.RequestLogger
		return reqLogger.ListenAndServeHTTP(ctx, ":9000", ":9002")
	})
	e := envoytest.NewEnvoyController(":8003")
	grp.Go("envoyController", func(ctx context.Context) error {
		return e.Run(ctx)
	})
	grp.Go("envoy", func(ctx context.Context) error {
		addr, err := envoytest.GetLoopbackAddr(ctx, 8003)
		if err != nil {
			return err
		}
		return envoytest.RunEnvoy(ctx, addr, "8080:8080")
	})
	grp.Go("downstream", func(ctx context.Context) error {
		d, err := makeDispatcher()
		if err != nil {
			return err
		}

		// One rule for each type of path match (exact, prefix, regex) and each type of header match
		// (exact and regex).
		if err := d.UpsertYaml(`
---
kind: Gateway
apiVersion: networking.x-k8s.io/v1alpha1
metadata:
  name: my-gateway
  namespace: default
spec:
  listeners:
  - protocol: HTTP
    port: 8080
---
kind: HTTPRoute
apiVersion: networking.x-k8s.io/v1alpha1
metadata:
  name: my-route
  namespace: default

spec:
  rules:
  - matches:
    - path:
        type: Exact
        value: /exact
    forwardTo:
    - serviceName: foo-backend-1
      port: 9000
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
`); err != nil {
			return err
		}

		loopbackIp, err := envoytest.GetLoopbackIp(ctx)
		if err != nil {
			return err
		}

		if err := d.Upsert(makeEndpoint("default", "foo-backend-1", loopbackIp, 9000)); err != nil {
			return err
		}
		if err := d.Upsert(makeEndpoint("default", "foo-backend-2", loopbackIp, 9001)); err != nil {
			return err
		}

		// Note: we know that the snapshot will not be nil, so no need to check. If it was then there is a
		// programming error with the test, so we are OK with it panic'ing and failing the test.
		version, snapshot := d.GetSnapshot(ctx)
		if status, err := e.Configure(ctx, "test-id", version, snapshot); err != nil {
			return err
		} else if status != nil {
			return fmt.Errorf("envoy error: %s", status.Message)
		}

		// Sometimes envoy seems to acknowledge the configuration before listening on the port. (This is
		// weird because sometimes envoy sends back an error indicating that it cannot bind to the
		// port. Either way, we need to check that we can actually connect before running the rest of
		// the tests.
		if err := checkReady(ctx, "http://127.0.0.1:8080/"); err != nil {
			return err
		}

		assertGet(&err, ctx, "http://127.0.0.1:8080/exact", 200, "Hello World")
		assertGet(&err, ctx, "http://127.0.0.1:8080/exact/foo", 404, "")
		assertGet(&err, ctx, "http://127.0.0.1:8080/prefix", 200, "Hello World")
		assertGet(&err, ctx, "http://127.0.0.1:8080/prefix/foo", 200, "Hello World")

		assertGet(&err, ctx, "http://127.0.0.1:8080/regular_expression", 200, "Hello World")
		assertGet(&err, ctx, "http://127.0.0.1:8080/regular_expression_a", 200, "Hello World")
		assertGet(&err, ctx, "http://127.0.0.1:8080/regular_expression_aaaaaaaa", 200, "Hello World")
		assertGet(&err, ctx, "http://127.0.0.1:8080/regular_expression_aaAaaaAa", 200, "Hello World")
		assertGet(&err, ctx, "http://127.0.0.1:8080/regular_expression_aaAaaaAab", 404, "")

		assertGetHeader(&err, ctx, "http://127.0.0.1:8080", "exact", "foo", 200, "Hello World")
		assertGetHeader(&err, ctx, "http://127.0.0.1:8080", "exact", "bar", 404, "")
		assertGetHeader(&err, ctx, "http://127.0.0.1:8080", "regular_expression", "foo", 200, "Hello World")
		assertGetHeader(&err, ctx, "http://127.0.0.1:8080", "regular_expression", "foo_aaaaAaaaa", 200, "Hello World")
		assertGetHeader(&err, ctx, "http://127.0.0.1:8080", "regular_expression", "foo_aaaaAaaaab", 404, "")
		assertGetHeader(&err, ctx, "http://127.0.0.1:8080", "regular_expression", "bar", 404, "")

		return err
	})
	assert.NoError(t, grp.Wait())
}

func TestBadMatchTypes(t *testing.T) {
	t.Parallel()
	d, err := makeDispatcher()
	require.NoError(t, err)

	// One rule for each type of path match (exact, prefix, regex) and each type of header match
	// (exact and regex).
	err = d.UpsertYaml(`
---
kind: HTTPRoute
apiVersion: networking.x-k8s.io/v1alpha1
metadata:
  name: my-route
  namespace: default
spec:
  rules:
  - matches:
    - path:
        type: Blah
        value: /exact
    forwardTo:
    - serviceName: foo-backend-1
      port: 9000
      weight: 100
`)
	assertErrorContains(t, err, `processing HTTPRoute:default:my-route: unknown path match type: "Blah"`)

	err = d.UpsertYaml(`
---
kind: HTTPRoute
apiVersion: networking.x-k8s.io/v1alpha1
metadata:
  name: my-route
  namespace: default
spec:
  rules:
  - matches:
    - headers:
        type: Bleh
        values:
          exact: foo
    forwardTo:
    - serviceName: foo-backend-1
      weight: 100
`)
	assertErrorContains(t, err, `processing HTTPRoute:default:my-route: unknown header match type: Bleh`)
}

func makeDispatcher() (*gateway.Dispatcher, error) {
	d := gateway.NewDispatcher()

	if err := d.Register("Gateway", func(untyped kates.Object) (*gateway.CompiledConfig, error) {
		return gateway.Compile_Gateway(untyped.(*gw.Gateway))
	}); err != nil {
		return nil, err
	}

	if err := d.Register("HTTPRoute", func(untyped kates.Object) (*gateway.CompiledConfig, error) {
		return gateway.Compile_HTTPRoute(untyped.(*gw.HTTPRoute))
	}); err != nil {
		return nil, err
	}

	if err := d.Register("Endpoints", func(untyped kates.Object) (*gateway.CompiledConfig, error) {
		return gateway.Compile_Endpoints(untyped.(*kates.Endpoints))
	}); err != nil {
		return nil, err
	}

	return d, nil
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

func checkReady(ctx context.Context, url string) error {
	delay := 10 * time.Millisecond
	for {
		if delay > 10*time.Second {
			return fmt.Errorf("url never became ready: %v", url)
		}
		_, err := http.Get(url)
		if err != nil {
			dlog.Infof(ctx, "error %v, retrying...", err)
			delay = delay * 2
			time.Sleep(delay)
			continue
		}
		return nil
	}
}

func get(ctx context.Context, url string, expectedCode int, expectedBody string, headers map[string]string) error {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return err
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	if resp.StatusCode != expectedCode {
		return fmt.Errorf("expected HTTP status code %d but got %d",
			expectedCode, resp.StatusCode)
	}
	actualBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if string(actualBody) != expectedBody {
		return fmt.Errorf("expected body %q but got %q",
			expectedBody, string(actualBody))
	}
	return nil
}

func assertGet(errPtr *error, ctx context.Context, url string, code int, expected string) {
	if *errPtr != nil {
		return
	}
	err := get(ctx, url, code, expected, nil)
	if err != nil && *errPtr == nil {
		*errPtr = err
	}
}

func assertGetHeader(errPtr *error, ctx context.Context, url, header, value string, code int, expected string) {
	if *errPtr != nil {
		return
	}
	err := get(ctx, url, code, expected, map[string]string{
		header: value,
	})
	if err != nil && *errPtr == nil {
		*errPtr = err
	}
}
