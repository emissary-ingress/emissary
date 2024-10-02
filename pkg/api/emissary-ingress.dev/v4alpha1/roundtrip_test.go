package v4alpha1_test

import (
	"encoding/json"
	"io/ioutil"
	"path"
	"testing"

	crds "github.com/emissary-ingress/emissary/v3/pkg/api/emissary-ingress.dev/v4alpha1"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"sigs.k8s.io/yaml"
)

func TestAuthSvcRoundTrip(t *testing.T) {
	var a []crds.AuthService
	checkRoundtrip(t, "authsvc.yaml", &a)
}

func TestDevPortalRoundTrip(t *testing.T) {
	var d []crds.DevPortal
	checkRoundtrip(t, "devportals.yaml", &d)
}

func TestHostRoundTrip(t *testing.T) {
	var h []crds.Host
	checkRoundtrip(t, "hosts.yaml", &h)
}

func TestLogSvcRoundTrip(t *testing.T) {
	var l []crds.LogService
	checkRoundtrip(t, "logsvc.yaml", &l)
}

func TestMappingRoundTrip(t *testing.T) {
	var m []crds.Mapping
	checkRoundtrip(t, "mappings.yaml", &m)
}

func TestModuleRoundTrip(t *testing.T) {
	var m []crds.Module
	checkRoundtrip(t, "modules.yaml", &m)
}

func TestRateLimitSvcRoundTrip(t *testing.T) {
	var r []crds.RateLimitService
	checkRoundtrip(t, "ratelimitsvc.yaml", &r)
}

func TestTCPMappingRoundTrip(t *testing.T) {
	var tm []crds.TCPMapping
	checkRoundtrip(t, "tcpmappings.yaml", &tm)
}

func TestTLSContextRoundTrip(t *testing.T) {
	var tc []crds.TLSContext
	checkRoundtrip(t, "tlscontexts.yaml", &tc)
}

func TestTracingSvcRoundTrip(t *testing.T) {
	var tr []crds.TracingService
	checkRoundtrip(t, "tracingsvc.yaml", &tr)
}

func checkRoundtrip(t *testing.T, filename string, ptr interface{}) {
	bytes, err := ioutil.ReadFile(path.Join("testdata", filename))
	require.NoError(t, err)

	canonical := func() string {
		var untyped interface{}
		require.NoError(t, yaml.Unmarshal(bytes, &untyped))
		canonical, err := json.MarshalIndent(untyped, "", "\t")
		require.NoError(t, err)
		return string(canonical)
	}()

	actual := func() string {
		// Round-trip twice, to get map field ordering, instead of struct field ordering.

		// first
		require.NoError(t, yaml.UnmarshalStrict(bytes, ptr))
		first, err := json.Marshal(ptr)
		require.NoError(t, err)

		// second
		var untyped interface{}
		require.NoError(t, json.Unmarshal(first, &untyped))
		second, err := json.MarshalIndent(untyped, "", "\t")
		require.NoError(t, err)

		return string(second)
	}()

	assert.Equal(t, canonical, actual)
}
