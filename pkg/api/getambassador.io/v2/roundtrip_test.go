package v2

import (
	"encoding/json"
	"io/ioutil"
	"path"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAmbassadorConfigRoundTrip(t *testing.T) {
	var m []Module
	checkRoundtrip(t, "modules.json", &m)
}

func TestMappingRoundTrip(t *testing.T) {
	var m []Mapping
	checkRoundtrip(t, "mappings.json", &m)
}

func checkRoundtrip(t *testing.T, filename string, ptr interface{}) {
	bytes, err := ioutil.ReadFile(path.Join("testdata", filename))
	require.NoError(t, err)

	err = json.Unmarshal(bytes, ptr)
	require.NoError(t, err)

	var canonical interface{}
	err = json.Unmarshal(bytes, &canonical)
	require.NoError(t, err)

	assert.Equal(t, canonical, roundtrip(ptr))
}

func roundtrip(obj interface{}) (result interface{}) {
	bytes, err := json.Marshal(obj)
	if err != nil {
		panic(err)
	}

	err = json.Unmarshal(bytes, &result)
	if err != nil {
		panic(err)
	}

	return
}
