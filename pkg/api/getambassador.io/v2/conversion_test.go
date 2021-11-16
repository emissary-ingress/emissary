package v2

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/datawire/ambassador/v2/pkg/api/getambassador.io/v3alpha1"
)

func bosNil() *BoolOrString {
	return &BoolOrString{}
}

func bosBool(b bool) *BoolOrString {
	return &BoolOrString{
		Bool: &b,
	}
}

func bosString(str string) *BoolOrString {
	return &BoolOrString{
		String: &str,
	}
}

func stringPtr(s string) *string {
	return &s
}

func TestConvert_v2_TLS_To_v3alpha1_TLS(t *testing.T) {
	t.Parallel()
	type subtest struct {
		inputTLS *BoolOrString
		inputSvc string

		expectedTLS            string
		expectedExplicitTLS    string
		expectedSvc            string
		expectedExplicitScheme *string
	}
	subtests := map[string]subtest{
		// falsey .tls
		//       {               inputs              /**/,             outputs                },
		"nn-ba": {nil /*************/, "svc" /**********/, "", "", "svc", nil},
		"nn-lh": {nil /*************/, "http://svc" /***/, "", "", "http://svc", nil},
		"nn-uh": {nil /*************/, "HTTP://svc" /***/, "", "", "HTTP://svc", nil},
		"nn-ls": {nil /*************/, "https://svc" /**/, "", "", "https://svc", nil},
		"nn-us": {nil /*************/, "HTTPS://svc" /**/, "", "", "HTTPS://svc", nil},
		"bn-ba": {bosNil() /********/, "svc" /**********/, "", "null", "svc", nil},
		"bn-lh": {bosNil() /********/, "http://svc" /***/, "", "null", "http://svc", nil},
		"bn-uh": {bosNil() /********/, "HTTP://svc" /***/, "", "null", "HTTP://svc", nil},
		"bn-ls": {bosNil() /********/, "https://svc" /**/, "", "null", "https://svc", nil},
		"bn-us": {bosNil() /********/, "HTTPS://svc" /**/, "", "null", "HTTPS://svc", nil},
		"bf-ba": {bosBool(false) /**/, "svc" /**********/, "", "bool:false", "svc", nil},
		"bf-lh": {bosBool(false) /**/, "http://svc" /***/, "", "bool:false", "http://svc", nil},
		"bf-uh": {bosBool(false) /**/, "HTTP://svc" /***/, "", "bool:false", "HTTP://svc", nil},
		"bf-ls": {bosBool(false) /**/, "https://svc" /**/, "", "bool:false", "https://svc", nil},
		"bf-us": {bosBool(false) /**/, "HTTPS://svc" /**/, "", "bool:false", "HTTPS://svc", nil},
		"be-ba": {bosString("") /***/, "svc" /**********/, "", "string", "svc", nil},
		"be-lh": {bosString("") /***/, "http://svc" /***/, "", "string", "http://svc", nil},
		"be-uh": {bosString("") /***/, "HTTP://svc" /***/, "", "string", "HTTP://svc", nil},
		"be-ls": {bosString("") /***/, "https://svc" /**/, "", "string", "https://svc", nil},
		"be-us": {bosString("") /***/, "HTTPS://svc" /**/, "", "string", "HTTPS://svc", nil},

		// truthy .tls
		//       {                inputs               /**/,               outputs                },
		"bt-ba": {bosBool(true) /*****/, "svc" /**********/, "", "bool:true", "https://svc", stringPtr("")},
		"bt-lh": {bosBool(true) /*****/, "http://svc" /***/, "", "bool:true", "https://svc", stringPtr("http://")},
		"bt-uh": {bosBool(true) /*****/, "HTTP://svc" /***/, "", "bool:true", "https://svc", stringPtr("HTTP://")},
		"bt-ls": {bosBool(true) /*****/, "https://svc" /**/, "", "bool:true", "https://svc", nil},
		"bt-hs": {bosBool(true) /*****/, "HTTPS://svc" /**/, "", "bool:true", "HTTPS://svc", nil},
		"bc-ba": {bosString("ctx") /**/, "svc" /**********/, "ctx", "", "svc", nil},
		"bc-lh": {bosString("ctx") /**/, "http://svc" /***/, "ctx", "", "http://svc", nil},
		"bc-uh": {bosString("ctx") /**/, "HTTP://svc" /***/, "ctx", "", "HTTP://svc", nil},
		"bc-ls": {bosString("ctx") /**/, "https://svc" /**/, "ctx", "", "https://svc", nil},
		"bc-hs": {bosString("ctx") /**/, "HTTPS://svc" /**/, "ctx", "", "HTTPS://svc", nil},
	}
	for name, info := range subtests {
		info := info // capture loop variable
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			var (
				actualTLS      string
				actualSvc      string
				actualExplicit *v3alpha1.V2ExplicitTLS
			)

			convert_v2_TLS_To_v3alpha1_TLS(
				info.inputTLS, info.inputSvc,
				&actualTLS, &actualSvc, &actualExplicit)
			assert.Equal(t, info.expectedTLS, actualTLS, "v2→v3alpha1 tls")
			assert.Equal(t, info.expectedSvc, actualSvc, "v2→v3alpha1 svc")
			if actualExplicit == nil {
				actualExplicit = &v3alpha1.V2ExplicitTLS{}
			}
			assert.Equal(t, info.expectedExplicitTLS, actualExplicit.TLS, "v2→v3alpha1 explicit.tls")
			assert.Equal(t, info.expectedExplicitScheme, actualExplicit.ServiceScheme, "v2→v3alpha1 explicit.svc")

			var (
				rtTLS *BoolOrString
				rtSvc string
			)
			convert_v3alpha1_TLS_To_v2_TLS(
				actualTLS, actualSvc, actualExplicit,
				&rtTLS, &rtSvc)
			assert.Equal(t, info.inputTLS, rtTLS, "v2→v3alpha1→v2 tls")
			assert.Equal(t, info.inputSvc, rtSvc, "v2→v3alpha1→v2 svc")
		})
	}
}

func TestConvert_v3alpha1_TLS_To_v2_TLS(t *testing.T) {
	t.Parallel()
	type subtest struct {
		inputTLS            string
		inputSvc            string
		inputExplicitTLS    string
		inputExplicitScheme *string

		expectedTLS *BoolOrString
		expectedSvc string
	}
	// Because TestConvert_v2_TLS_To_v3alpha1_TLS tests round-trips, we only need to test the
	// cases where one of the explicit fields is *ignored*.  This can happen if a resource that
	// was originally v2 is later edited as v3alpha1.
	subtests := map[string]subtest{
		// empty
		"minimal":   { /*input:*/ "", "svc", "", nil /*output:*/, nil, "svc"},
		"http":      { /*input:*/ "", "HTTP://svc", "", nil /*output:*/, nil, "HTTP://svc"},
		"https":     { /*input:*/ "", "HTTPS://svc", "", nil /*output:*/, nil, "HTTPS://svc"},
		"http-ctx":  { /*input:*/ "ctx", "HTTP://svc", "", nil /*output:*/, bosString("ctx"), "HTTP://svc"},
		"https-ctx": { /*input:*/ "ctx", "HTTPS://svc", "", nil /*output:*/, bosString("ctx"), "HTTPS://svc"},

		// ignore explicitTLS:
		"ignoretls-n":  { /*input:*/ "ctx", "HTTP://svc", "null", nil /*output:*/, bosString("ctx"), "HTTP://svc"},
		"ignoretls-f":  { /*input:*/ "ctx", "HTTP://svc", "bool:false", nil /*output:*/, bosString("ctx"), "HTTP://svc"},
		"ignoretls-t":  { /*input:*/ "ctx", "HTTP://svc", "bool:true", nil /*output:*/, bosString("ctx"), "HTTP://svc"},
		"ignoretls-s":  { /*input:*/ "ctx", "HTTP://svc", "string", nil /*output:*/, bosString("ctx"), "HTTP://svc"},
		"ignoretls-t2": { /*input:*/ "", "HTTP://svc", "bool:true", nil /*output:*/, nil, "HTTP://svc"},

		// ignore explicitSvc
		"ignoresvc-http":  { /*input:*/ "", "HTTP://svc", "", stringPtr("https://") /*output:*/, nil, "HTTP://svc"},
		"ignoresvc-https": { /*input:*/ "", "HTTPS://svc", "", stringPtr("http://") /*output:*/, nil, "HTTPS://svc"},
	}
	for name, info := range subtests {
		info := info // capture loop variable
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			var (
				actualTLS *BoolOrString
				actualSvc string
			)
			convert_v3alpha1_TLS_To_v2_TLS(
				info.inputTLS, info.inputSvc,
				&v3alpha1.V2ExplicitTLS{
					TLS:           info.inputExplicitTLS,
					ServiceScheme: info.inputExplicitScheme,
				},
				&actualTLS, &actualSvc)
			assert.Equal(t, info.expectedTLS, actualTLS, "v3alpha1→v2 tls")
			assert.Equal(t, info.expectedSvc, actualSvc, "v3alpha1→v2 svc")

			var (
				rtTLS      string
				rtSvc      string
				rtExplicit *v3alpha1.V2ExplicitTLS
			)
			convert_v2_TLS_To_v3alpha1_TLS(
				actualTLS, actualSvc,
				&rtTLS, &rtSvc, &rtExplicit)
			if rtExplicit == nil {
				rtExplicit = &v3alpha1.V2ExplicitTLS{}
			}
			assert.Equal(t, info.inputTLS, rtTLS, "v3alpha1→v2→v3alpha1 tls")
			assert.Equal(t, info.inputSvc, rtSvc, "v3alpha1→v2→v3alpha1 svc")
			if !strings.Contains(t.Name(), "ignoretls") {
				assert.Equal(t, info.inputExplicitTLS, rtExplicit.TLS, "v3alpha1→v2→v3alpha1 explicit.tls")
			}
			if !strings.Contains(t.Name(), "ignoresvc") {
				assert.Equal(t, info.inputExplicitScheme, rtExplicit.ServiceScheme, "v3alpha1→v2→v3alpha1 explicit.svc")
			}
		})
	}
}
