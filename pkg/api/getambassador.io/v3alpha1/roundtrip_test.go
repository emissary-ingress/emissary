package v3alpha1

// Note that getambassador.io/v2 contains many more CRDs than does
// getambassador.io/v3alpha1. This test only covers the CRDs that are
// only in v3alpha1 -- v2 resources are tested by their own roundtrip
// test.

import (
	"encoding/json"
	"io/ioutil"
	"path"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Need AmbassadorListener and AmbassadorHost too...

func TestMappingRoundTrip(t *testing.T) {
	var m []AmbassadorMapping
	checkRoundtrip(t, "mappings.json", &m)
}

func TestTCPMappingRoundTrip(t *testing.T) {
	var tm []AmbassadorTCPMapping
	checkRoundtrip(t, "tcpmappings.json", &tm)
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
