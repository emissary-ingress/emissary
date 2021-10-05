package v3alpha1

import (
	"encoding/json"
	"io/ioutil"
	"path"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAuthSvcRoundTrip(t *testing.T) {
	var a []AuthService
	checkRoundtrip(t, "authsvc.json", &a)
}

func TestDevPortalRoundTrip(t *testing.T) {
	var d []DevPortal
	checkRoundtrip(t, "devportals.json", &d)
}

func TestHostRoundTrip(t *testing.T) {
	var h []Host
	checkRoundtrip(t, "hosts.json", &h)
}

func TestLogSvcRoundTrip(t *testing.T) {
	var l []LogService
	checkRoundtrip(t, "logsvc.json", &l)
}

func TestMappingRoundTrip(t *testing.T) {
	var m []Mapping
	checkRoundtrip(t, "mappings.json", &m)
}

func TestModuleRoundTrip(t *testing.T) {
	var m []Module
	checkRoundtrip(t, "modules.json", &m)
}

func TestRateLimitSvcRoundTrip(t *testing.T) {
	var r []RateLimitService
	checkRoundtrip(t, "ratelimitsvc.json", &r)
}

func TestTCPMappingRoundTrip(t *testing.T) {
	var tm []TCPMapping
	checkRoundtrip(t, "tcpmappings.json", &tm)
}

func TestTLSContextRoundTrip(t *testing.T) {
	var tc []TLSContext
	checkRoundtrip(t, "tlscontexts.json", &tc)
}

func TestTracingSvcRoundTrip(t *testing.T) {
	var tr []TracingService
	checkRoundtrip(t, "tracingsvc.json", &tr)
}

func checkRoundtrip(t *testing.T, filename string, ptr interface{}) {
	bytes, err := ioutil.ReadFile(path.Join("testdata", filename))
	require.NoError(t, err)

	canonical := func() string {
		var untyped interface{}
		err = json.Unmarshal(bytes, &untyped)
		require.NoError(t, err)
		canonical, err := json.MarshalIndent(untyped, "", "\t")
		require.NoError(t, err)
		return string(canonical)
	}()

	actual := func() string {
		// Round-trip twice, to get map field ordering, instead of struct field ordering.

		// first
		err = json.Unmarshal(bytes, ptr)
		require.NoError(t, err)
		first, err := json.Marshal(ptr)
		require.NoError(t, err)

		// second
		var untyped interface{}
		err = json.Unmarshal(first, &untyped)
		require.NoError(t, err)
		second, err := json.MarshalIndent(untyped, "", "\t")
		require.NoError(t, err)

		return string(second)
	}()

	assert.Equal(t, canonical, actual)
}
