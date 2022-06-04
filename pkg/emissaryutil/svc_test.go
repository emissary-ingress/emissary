package emissaryutil

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseServiceName(t *testing.T) {
	t.Parallel()
	testcases := map[string]struct {
		Scheme   string
		Hostname string
		Port     uint16
		Err      string
	}{
		"[fe80::e022:9cff:fecc:c7c4%zone]":   {"", "", 0, `service "[fe80::e022:9cff:fecc:c7c4%zone]": parse "//[fe80::e022:9cff:fecc:c7c4%zone]": invalid URL escape "%zo"`},
		"[fe80::e022:9cff:fecc:c7c4%25zone]": {"", "fe80::e022:9cff:fecc:c7c4%zone", 0, ``},
		"https://[::1%25lo]:443":             {"https", "::1%lo", 443, ``},
	}
	for input, exp := range testcases {
		input := input
		exp := exp
		t.Run(input, func(t *testing.T) {
			t.Parallel()
			scheme, hostname, port, err := ParseServiceName(input)
			assert.Equal(t, exp.Scheme, scheme)
			assert.Equal(t, exp.Hostname, hostname)
			assert.Equal(t, exp.Port, port)
			if exp.Err == "" {
				assert.NoError(t, err)
			} else {
				assert.EqualError(t, err, exp.Err)
			}
		})
	}
}

type testGlobalResolverConfig struct {
	ambassadorNamespace                        string
	useAmbassadorNamespaceForServiceResolution bool
}

func (ir testGlobalResolverConfig) AmbassadorNamespace() string { return ir.ambassadorNamespace }
func (ir testGlobalResolverConfig) UseAmbassadorNamespaceForServiceResolution() bool {
	return ir.useAmbassadorNamespaceForServiceResolution
}

// TestQualifyService mimics python/tests/unit/test_qualify_service.py:test_qualify_service().
// Please keep them in-sync.
func TestQualifyService(t *testing.T) {
	t.Parallel()

	type tcInput struct {
		Svc          string
		Namespace    string
		ResolverKind string
	}
	type tc struct {
		Input  tcInput
		Output string
		Err    string
	}
	normalizeServiceName := func(service, namespace, resolverKind string) tcInput {
		return tcInput{
			Svc:          service,
			Namespace:    namespace,
			ResolverKind: resolverKind,
		}
	}
	qualifyServiceName := func(service, namespace string) tcInput {
		return tcInput{
			Svc:          service,
			Namespace:    namespace,
			ResolverKind: "KubernetesTestResolver",
		}
	}
	ir := testGlobalResolverConfig{
		ambassadorNamespace:                        "default",
		useAmbassadorNamespaceForServiceResolution: false,
	}
	testcases := []tc{
		{Input: qualifyServiceName("backoffice", ""), Output: "backoffice"},
		{Input: qualifyServiceName("backoffice", "default"), Output: "backoffice"},
		{Input: qualifyServiceName("backoffice", "otherns"), Output: "backoffice.otherns"},
		{Input: qualifyServiceName("backoffice.otherns", ""), Output: "backoffice.otherns"},
		{Input: qualifyServiceName("backoffice.otherns", "default"), Output: "backoffice.otherns"},
		{Input: qualifyServiceName("backoffice.otherns", "otherns"), Output: "backoffice.otherns"},

		{Input: normalizeServiceName("backoffice", "", "ConsulResolver"), Output: "backoffice"},
		{Input: normalizeServiceName("backoffice", "default", "ConsulResolver"), Output: "backoffice"},
		{Input: normalizeServiceName("backoffice", "otherns", "ConsulResolver"), Output: "backoffice"},
		{Input: normalizeServiceName("backoffice.otherns", "", "ConsulResolver"), Output: "backoffice.otherns"},
		{Input: normalizeServiceName("backoffice.otherns", "default", "ConsulResolver"), Output: "backoffice.otherns"},
		{Input: normalizeServiceName("backoffice.otherns", "otherns", "ConsulResolver"), Output: "backoffice.otherns"},

		{Input: qualifyServiceName("backoffice:80", ""), Output: "backoffice:80"},
		{Input: qualifyServiceName("backoffice:80", "default"), Output: "backoffice:80"},
		{Input: qualifyServiceName("backoffice:80", "otherns"), Output: "backoffice.otherns:80"},
		{Input: qualifyServiceName("backoffice.otherns:80", ""), Output: "backoffice.otherns:80"},
		{Input: qualifyServiceName("backoffice.otherns:80", "default"), Output: "backoffice.otherns:80"},
		{Input: qualifyServiceName("backoffice.otherns:80", "otherns"), Output: "backoffice.otherns:80"},

		{Input: qualifyServiceName("[fe80::e022:9cff:fecc:c7c4]", ""), Output: "[fe80::e022:9cff:fecc:c7c4]"},
		{Input: qualifyServiceName("[fe80::e022:9cff:fecc:c7c4]", "default"), Output: "[fe80::e022:9cff:fecc:c7c4]"},
		{Input: qualifyServiceName("[fe80::e022:9cff:fecc:c7c4]", "other"), Output: "[fe80::e022:9cff:fecc:c7c4]"},
		{Input: qualifyServiceName("https://[fe80::e022:9cff:fecc:c7c4]", ""), Output: "https://[fe80::e022:9cff:fecc:c7c4]"},
		{Input: qualifyServiceName("https://[fe80::e022:9cff:fecc:c7c4]", "default"), Output: "https://[fe80::e022:9cff:fecc:c7c4]"},
		{Input: qualifyServiceName("https://[fe80::e022:9cff:fecc:c7c4]", "other"), Output: "https://[fe80::e022:9cff:fecc:c7c4]"},
		{Input: qualifyServiceName("https://[fe80::e022:9cff:fecc:c7c4]:443", ""), Output: "https://[fe80::e022:9cff:fecc:c7c4]:443"},
		{Input: qualifyServiceName("https://[fe80::e022:9cff:fecc:c7c4]:443", "default"), Output: "https://[fe80::e022:9cff:fecc:c7c4]:443"},
		{Input: qualifyServiceName("https://[fe80::e022:9cff:fecc:c7c4]:443", "other"), Output: "https://[fe80::e022:9cff:fecc:c7c4]:443"},
		{Input: qualifyServiceName("https://[fe80::e022:9cff:fecc:c7c4%25zone]:443", "other"), Output: "https://[fe80::e022:9cff:fecc:c7c4%25zone]:443"},

		{Input: normalizeServiceName("backoffice:80", "", "ConsulResolver"), Output: "backoffice:80"},
		{Input: normalizeServiceName("backoffice:80", "default", "ConsulResolver"), Output: "backoffice:80"},
		{Input: normalizeServiceName("backoffice:80", "otherns", "ConsulResolver"), Output: "backoffice:80"},
		{Input: normalizeServiceName("backoffice.otherns:80", "", "ConsulResolver"), Output: "backoffice.otherns:80"},
		{Input: normalizeServiceName("backoffice.otherns:80", "default", "ConsulResolver"), Output: "backoffice.otherns:80"},
		{Input: normalizeServiceName("backoffice.otherns:80", "otherns", "ConsulResolver"), Output: "backoffice.otherns:80"},

		{Input: qualifyServiceName("http://backoffice", ""), Output: "http://backoffice"},
		{Input: qualifyServiceName("http://backoffice", "default"), Output: "http://backoffice"},
		{Input: qualifyServiceName("http://backoffice", "otherns"), Output: "http://backoffice.otherns"},
		{Input: qualifyServiceName("http://backoffice.otherns", ""), Output: "http://backoffice.otherns"},
		{Input: qualifyServiceName("http://backoffice.otherns", "default"), Output: "http://backoffice.otherns"},
		{Input: qualifyServiceName("http://backoffice.otherns", "otherns"), Output: "http://backoffice.otherns"},

		{Input: normalizeServiceName("http://backoffice", "", "ConsulResolver"), Output: "http://backoffice"},
		{Input: normalizeServiceName("http://backoffice", "default", "ConsulResolver"), Output: "http://backoffice"},
		{Input: normalizeServiceName("http://backoffice", "otherns", "ConsulResolver"), Output: "http://backoffice"},
		{Input: normalizeServiceName("http://backoffice.otherns", "", "ConsulResolver"), Output: "http://backoffice.otherns"},
		{Input: normalizeServiceName("http://backoffice.otherns", "default", "ConsulResolver"), Output: "http://backoffice.otherns"},
		{Input: normalizeServiceName("http://backoffice.otherns", "otherns", "ConsulResolver"), Output: "http://backoffice.otherns"},

		{Input: qualifyServiceName("http://backoffice:80", ""), Output: "http://backoffice:80"},
		{Input: qualifyServiceName("http://backoffice:80", "default"), Output: "http://backoffice:80"},
		{Input: qualifyServiceName("http://backoffice:80", "otherns"), Output: "http://backoffice.otherns:80"},
		{Input: qualifyServiceName("http://backoffice.otherns:80", ""), Output: "http://backoffice.otherns:80"},
		{Input: qualifyServiceName("http://backoffice.otherns:80", "default"), Output: "http://backoffice.otherns:80"},
		{Input: qualifyServiceName("http://backoffice.otherns:80", "otherns"), Output: "http://backoffice.otherns:80"},

		{Input: normalizeServiceName("http://backoffice:80", "", "ConsulResolver"), Output: "http://backoffice:80"},
		{Input: normalizeServiceName("http://backoffice:80", "default", "ConsulResolver"), Output: "http://backoffice:80"},
		{Input: normalizeServiceName("http://backoffice:80", "otherns", "ConsulResolver"), Output: "http://backoffice:80"},
		{Input: normalizeServiceName("http://backoffice.otherns:80", "", "ConsulResolver"), Output: "http://backoffice.otherns:80"},
		{Input: normalizeServiceName("http://backoffice.otherns:80", "default", "ConsulResolver"), Output: "http://backoffice.otherns:80"},
		{Input: normalizeServiceName("http://backoffice.otherns:80", "otherns", "ConsulResolver"), Output: "http://backoffice.otherns:80"},

		{Input: qualifyServiceName("https://backoffice", ""), Output: "https://backoffice"},
		{Input: qualifyServiceName("https://backoffice", "default"), Output: "https://backoffice"},
		{Input: qualifyServiceName("https://backoffice", "otherns"), Output: "https://backoffice.otherns"},
		{Input: qualifyServiceName("https://backoffice.otherns", ""), Output: "https://backoffice.otherns"},
		{Input: qualifyServiceName("https://backoffice.otherns", "default"), Output: "https://backoffice.otherns"},
		{Input: qualifyServiceName("https://backoffice.otherns", "otherns"), Output: "https://backoffice.otherns"},

		{Input: normalizeServiceName("https://backoffice", "", "ConsulResolver"), Output: "https://backoffice"},
		{Input: normalizeServiceName("https://backoffice", "default", "ConsulResolver"), Output: "https://backoffice"},
		{Input: normalizeServiceName("https://backoffice", "otherns", "ConsulResolver"), Output: "https://backoffice"},
		{Input: normalizeServiceName("https://backoffice.otherns", "", "ConsulResolver"), Output: "https://backoffice.otherns"},
		{Input: normalizeServiceName("https://backoffice.otherns", "default", "ConsulResolver"), Output: "https://backoffice.otherns"},
		{Input: normalizeServiceName("https://backoffice.otherns", "otherns", "ConsulResolver"), Output: "https://backoffice.otherns"},

		{Input: qualifyServiceName("https://backoffice:443", ""), Output: "https://backoffice:443"},
		{Input: qualifyServiceName("https://backoffice:443", "default"), Output: "https://backoffice:443"},
		{Input: qualifyServiceName("https://backoffice:443", "otherns"), Output: "https://backoffice.otherns:443"},
		{Input: qualifyServiceName("https://backoffice.otherns:443", ""), Output: "https://backoffice.otherns:443"},
		{Input: qualifyServiceName("https://backoffice.otherns:443", "default"), Output: "https://backoffice.otherns:443"},
		{Input: qualifyServiceName("https://backoffice.otherns:443", "otherns"), Output: "https://backoffice.otherns:443"},

		{Input: normalizeServiceName("https://backoffice:443", "", "ConsulResolver"), Output: "https://backoffice:443"},
		{Input: normalizeServiceName("https://backoffice:443", "default", "ConsulResolver"), Output: "https://backoffice:443"},
		{Input: normalizeServiceName("https://backoffice:443", "otherns", "ConsulResolver"), Output: "https://backoffice:443"},
		{Input: normalizeServiceName("https://backoffice.otherns:443", "", "ConsulResolver"), Output: "https://backoffice.otherns:443"},
		{Input: normalizeServiceName("https://backoffice.otherns:443", "default", "ConsulResolver"), Output: "https://backoffice.otherns:443"},
		{Input: normalizeServiceName("https://backoffice.otherns:443", "otherns", "ConsulResolver"), Output: "https://backoffice.otherns:443"},

		{Input: qualifyServiceName("localhost", ""), Output: "localhost"},
		{Input: qualifyServiceName("localhost", "default"), Output: "localhost"},
		{Input: qualifyServiceName("localhost", "otherns"), Output: "localhost"},
		// It's not meaningful to actually say "localhost.otherns", but it should passed through unchanged.
		{Input: qualifyServiceName("localhost.otherns", ""), Output: "localhost.otherns"},
		{Input: qualifyServiceName("localhost.otherns", "default"), Output: "localhost.otherns"},
		{Input: qualifyServiceName("localhost.otherns", "otherns"), Output: "localhost.otherns"},

		{Input: normalizeServiceName("localhost", "", "ConsulResolver"), Output: "localhost"},
		{Input: normalizeServiceName("localhost", "default", "ConsulResolver"), Output: "localhost"},
		{Input: normalizeServiceName("localhost", "otherns", "ConsulResolver"), Output: "localhost"},
		// It's not meaningful to actually say "localhost.otherns", but it should passed through unchanged.
		{Input: normalizeServiceName("localhost.otherns", "", "ConsulResolver"), Output: "localhost.otherns"},
		{Input: normalizeServiceName("localhost.otherns", "default", "ConsulResolver"), Output: "localhost.otherns"},
		{Input: normalizeServiceName("localhost.otherns", "otherns", "ConsulResolver"), Output: "localhost.otherns"},

		{Input: qualifyServiceName("localhost:80", ""), Output: "localhost:80"},
		{Input: qualifyServiceName("localhost:80", "default"), Output: "localhost:80"},
		{Input: qualifyServiceName("localhost:80", "otherns"), Output: "localhost:80"},
		// It's not meaningful to actually say "localhost.otherns", but it should passed through unchanged.
		{Input: qualifyServiceName("localhost.otherns:80", ""), Output: "localhost.otherns:80"},
		{Input: qualifyServiceName("localhost.otherns:80", "default"), Output: "localhost.otherns:80"},
		{Input: qualifyServiceName("localhost.otherns:80", "otherns"), Output: "localhost.otherns:80"},

		{Input: normalizeServiceName("localhost:80", "", "ConsulResolver"), Output: "localhost:80"},
		{Input: normalizeServiceName("localhost:80", "default", "ConsulResolver"), Output: "localhost:80"},
		{Input: normalizeServiceName("localhost:80", "otherns", "ConsulResolver"), Output: "localhost:80"},
		// It's not meaningful to actually say "localhost.otherns", but it should passed through unchanged.
		{Input: normalizeServiceName("localhost.otherns:80", "", "ConsulResolver"), Output: "localhost.otherns:80"},
		{Input: normalizeServiceName("localhost.otherns:80", "default", "ConsulResolver"), Output: "localhost.otherns:80"},
		{Input: normalizeServiceName("localhost.otherns:80", "otherns", "ConsulResolver"), Output: "localhost.otherns:80"},

		{Input: qualifyServiceName("http://localhost", ""), Output: "http://localhost"},
		{Input: qualifyServiceName("http://localhost", "default"), Output: "http://localhost"},
		{Input: qualifyServiceName("http://localhost", "otherns"), Output: "http://localhost"},
		// It's not meaningful to actually say "localhost.otherns", but it should passed through unchanged.
		{Input: qualifyServiceName("http://localhost.otherns", ""), Output: "http://localhost.otherns"},
		{Input: qualifyServiceName("http://localhost.otherns", "default"), Output: "http://localhost.otherns"},
		{Input: qualifyServiceName("http://localhost.otherns", "otherns"), Output: "http://localhost.otherns"},

		{Input: normalizeServiceName("http://localhost", "", "ConsulResolver"), Output: "http://localhost"},
		{Input: normalizeServiceName("http://localhost", "default", "ConsulResolver"), Output: "http://localhost"},
		{Input: normalizeServiceName("http://localhost", "otherns", "ConsulResolver"), Output: "http://localhost"},
		// It's not meaningful to actually say "localhost.otherns", but it should passed through unchanged.
		{Input: normalizeServiceName("http://localhost.otherns", "", "ConsulResolver"), Output: "http://localhost.otherns"},
		{Input: normalizeServiceName("http://localhost.otherns", "default", "ConsulResolver"), Output: "http://localhost.otherns"},
		{Input: normalizeServiceName("http://localhost.otherns", "otherns", "ConsulResolver"), Output: "http://localhost.otherns"},

		{Input: qualifyServiceName("http://localhost:80", ""), Output: "http://localhost:80"},
		{Input: qualifyServiceName("http://localhost:80", "default"), Output: "http://localhost:80"},
		{Input: qualifyServiceName("http://localhost:80", "otherns"), Output: "http://localhost:80"},
		// It's not meaningful to actually say "localhost.otherns", but it should passed through unchanged.
		{Input: qualifyServiceName("http://localhost.otherns:80", ""), Output: "http://localhost.otherns:80"},
		{Input: qualifyServiceName("http://localhost.otherns:80", "default"), Output: "http://localhost.otherns:80"},
		{Input: qualifyServiceName("http://localhost.otherns:80", "otherns"), Output: "http://localhost.otherns:80"},

		{Input: normalizeServiceName("http://localhost:80", "", "ConsulResolver"), Output: "http://localhost:80"},
		{Input: normalizeServiceName("http://localhost:80", "default", "ConsulResolver"), Output: "http://localhost:80"},
		{Input: normalizeServiceName("http://localhost:80", "otherns", "ConsulResolver"), Output: "http://localhost:80"},
		// It's not meaningful to actually say "localhost.otherns", but it should passed through unchanged.
		{Input: normalizeServiceName("http://localhost.otherns:80", "", "ConsulResolver"), Output: "http://localhost.otherns:80"},
		{Input: normalizeServiceName("http://localhost.otherns:80", "default", "ConsulResolver"), Output: "http://localhost.otherns:80"},
		{Input: normalizeServiceName("http://localhost.otherns:80", "otherns", "ConsulResolver"), Output: "http://localhost.otherns:80"},

		{Input: qualifyServiceName("https://localhost", ""), Output: "https://localhost"},
		{Input: qualifyServiceName("https://localhost", "default"), Output: "https://localhost"},
		{Input: qualifyServiceName("https://localhost", "otherns"), Output: "https://localhost"},
		// It's not meaningful to actually say "localhost.otherns", but it should passed through unchanged.
		{Input: qualifyServiceName("https://localhost.otherns", ""), Output: "https://localhost.otherns"},
		{Input: qualifyServiceName("https://localhost.otherns", "default"), Output: "https://localhost.otherns"},
		{Input: qualifyServiceName("https://localhost.otherns", "otherns"), Output: "https://localhost.otherns"},

		{Input: normalizeServiceName("https://localhost", "", "ConsulResolver"), Output: "https://localhost"},
		{Input: normalizeServiceName("https://localhost", "default", "ConsulResolver"), Output: "https://localhost"},
		{Input: normalizeServiceName("https://localhost", "otherns", "ConsulResolver"), Output: "https://localhost"},
		// It's not meaningful to actually say "localhost.otherns", but it should passed through unchanged.
		{Input: normalizeServiceName("https://localhost.otherns", "", "ConsulResolver"), Output: "https://localhost.otherns"},
		{Input: normalizeServiceName("https://localhost.otherns", "default", "ConsulResolver"), Output: "https://localhost.otherns"},
		{Input: normalizeServiceName("https://localhost.otherns", "otherns", "ConsulResolver"), Output: "https://localhost.otherns"},

		{Input: qualifyServiceName("https://localhost:443", ""), Output: "https://localhost:443"},
		{Input: qualifyServiceName("https://localhost:443", "default"), Output: "https://localhost:443"},
		{Input: qualifyServiceName("https://localhost:443", "otherns"), Output: "https://localhost:443"},
		// It's not meaningful to actually say "localhost.otherns", but it should passed through unchanged.
		{Input: qualifyServiceName("https://localhost.otherns:443", ""), Output: "https://localhost.otherns:443"},
		{Input: qualifyServiceName("https://localhost.otherns:443", "default"), Output: "https://localhost.otherns:443"},
		{Input: qualifyServiceName("https://localhost.otherns:443", "otherns"), Output: "https://localhost.otherns:443"},

		{Input: normalizeServiceName("https://localhost:443", "", "ConsulResolver"), Output: "https://localhost:443"},
		{Input: normalizeServiceName("https://localhost:443", "default", "ConsulResolver"), Output: "https://localhost:443"},
		{Input: normalizeServiceName("https://localhost:443", "otherns", "ConsulResolver"), Output: "https://localhost:443"},
		// It's not meaningful to actually say "localhost.otherns", but it should passed through unchanged.
		{Input: normalizeServiceName("https://localhost.otherns:443", "", "ConsulResolver"), Output: "https://localhost.otherns:443"},
		{Input: normalizeServiceName("https://localhost.otherns:443", "default", "ConsulResolver"), Output: "https://localhost.otherns:443"},
		{Input: normalizeServiceName("https://localhost.otherns:443", "otherns", "ConsulResolver"), Output: "https://localhost.otherns:443"},

		{Input: qualifyServiceName("ambassador://foo.ns", "otherns"), Output: "ambassador://foo.ns"}, // let's not introduce silly semantics
		{Input: qualifyServiceName("//foo.ns:1234", "otherns"), Output: "foo.ns:1234"},               // we tell people "URL-ish", actually support URL-ish
		{Input: qualifyServiceName("foo.ns:1234", "otherns"), Output: "foo.ns:1234"},

		{Input: normalizeServiceName("ambassador://foo.ns", "otherns", "ConsulResolver"), Output: "ambassador://foo.ns"}, // let's not introduce silly semantics
		{Input: normalizeServiceName("//foo.ns:1234", "otherns", "ConsulResolver"), Output: "foo.ns:1234"},               // we tell people "URL-ish", actually support URL-ish
		{Input: normalizeServiceName("foo.ns:1234", "otherns", "ConsulResolver"), Output: "foo.ns:1234"},

		{Input: qualifyServiceName("https://bad-service:443:443", "otherns"), Err: `service "https://bad-service:443:443": address bad-service:443:443: too many colons in address`},
		{Input: qualifyServiceName("bad-service:443:443", "otherns"), Err: `service "bad-service:443:443": address bad-service:443:443: too many colons in address`},
		{Input: qualifyServiceName("https://[fe80::e022:9cff:fecc:c7c4:443", "otherns"), Err: `service "https://[fe80::e022:9cff:fecc:c7c4:443": parse "https://[fe80::e022:9cff:fecc:c7c4:443": missing ']' in host`},
		{Input: qualifyServiceName("https://[fe80::e022:9cff:fecc:c7c4", "otherns"), Err: `service "https://[fe80::e022:9cff:fecc:c7c4": parse "https://[fe80::e022:9cff:fecc:c7c4": missing ']' in host`},
		{Input: qualifyServiceName("https://fe80::e022:9cff:fecc:c7c4", "otherns"), Err: `service "https://fe80::e022:9cff:fecc:c7c4": parse "https://fe80::e022:9cff:fecc:c7c4": invalid port ":c7c4" after host`},
		{Input: qualifyServiceName("https://bad-service:-1", "otherns"), Err: `service "https://bad-service:-1": parse "https://bad-service:-1": invalid port ":-1" after host`},
		{Input: qualifyServiceName("https://bad-service:70000", "otherns"), Err: `service "https://bad-service:70000": port 70000: strconv.ParseUint: parsing "70000": value out of range`},

		{Input: normalizeServiceName("https://bad-service:443:443", "otherns", "ConsulResolver"), Err: `service "https://bad-service:443:443": address bad-service:443:443: too many colons in address`},
		{Input: normalizeServiceName("bad-service:443:443", "otherns", "ConsulResolver"), Err: `service "bad-service:443:443": address bad-service:443:443: too many colons in address`},
		{Input: normalizeServiceName("https://[fe80::e022:9cff:fecc:c7c4:443", "otherns", "ConsulResolver"), Err: `service "https://[fe80::e022:9cff:fecc:c7c4:443": parse "https://[fe80::e022:9cff:fecc:c7c4:443": missing ']' in host`},
		{Input: normalizeServiceName("https://[fe80::e022:9cff:fecc:c7c4", "otherns", "ConsulResolver"), Err: `service "https://[fe80::e022:9cff:fecc:c7c4": parse "https://[fe80::e022:9cff:fecc:c7c4": missing ']' in host`},
		{Input: normalizeServiceName("https://fe80::e022:9cff:fecc:c7c4", "otherns", "ConsulResolver"), Err: `service "https://fe80::e022:9cff:fecc:c7c4": parse "https://fe80::e022:9cff:fecc:c7c4": invalid port ":c7c4" after host`},
		{Input: normalizeServiceName("https://bad-service:-1", "otherns", "ConsulResolver"), Err: `service "https://bad-service:-1": parse "https://bad-service:-1": invalid port ":-1" after host`},
		{Input: normalizeServiceName("https://bad-service:70000", "otherns", "ConsulResolver"), Err: `service "https://bad-service:70000": port 70000: strconv.ParseUint: parsing "70000": value out of range`},
		{Input: qualifyServiceName("https://[fe80::e022:9cff:fecc:c7c4%zone]:443", "other"), Err: `service "https://[fe80::e022:9cff:fecc:c7c4%zone]:443": parse "https://[fe80::e022:9cff:fecc:c7c4%zone]:443": invalid URL escape "%zo"`},
	}
	for i, tc := range testcases {
		tc := tc // capture loop variable
		t.Run(fmt.Sprintf("%v: %v", i, tc), func(t *testing.T) {
			t.Parallel()
			actVal, actErr := NormalizeServiceName(ir, tc.Input.Svc, tc.Input.Namespace, tc.Input.ResolverKind)
			if tc.Err == "" {
				assert.Equal(t, tc.Output, actVal)
				assert.NoError(t, actErr)
			} else {
				assert.Equal(t, "", actVal)
				assert.EqualError(t, actErr, tc.Err)
			}
		})
	}
}

func TestIsLocalhost(t *testing.T) {
	t.Parallel()
	testcases := map[string]bool{
		"localhost":  true,
		"localhost4": false, // don't rely on non-standard things sometimes in /etc/hosts
		"localhost6": false, // don't rely on non-standard things sometimes in /etc/hosts

		"127.0.0.1":  true,
		"127.0.0.01": false, // RFC 3986 URL parsing says to not treat it as an IPv4 if there are leading zeros (but is what we consume a URL?)
		"2130706433": false, // RFC 3986 URL parsing says to not treat it as an IPv4 unless it's dotted-quad (but is what we consume a URL?)
		"127.1":      false, // RFC 3986 URL parsing says to not treat it as an IPv4 unless it's dotted-quad (but is what we consume a URL?)

		"127.2.3.4": true,

		"::1":        true,
		"::1%lo":     true,
		"::0:0:1":    true,
		"::00:00:01": true,
	}
	for hostname, exp := range testcases {
		hostname := hostname
		exp := exp
		t.Run(hostname, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, exp, IsLocalhost(hostname))
		})
	}
}
