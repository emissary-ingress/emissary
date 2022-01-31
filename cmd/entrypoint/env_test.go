package entrypoint

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetEnvoyFlags(t *testing.T) {
	foundFlag := false
	foundValue := false

	os.Setenv("ENVOY_CONCURRENCY", "4")

	flags := GetEnvoyFlags()
	for idx, flag := range flags {
		if flag == "--concurrency" {
			foundFlag = true
			t.Logf("flags[idx] = %v", flags[idx])
			if idx+1 < len(flags) && flags[idx+1] == "4" {
				foundValue = true
			}
			break
		}
	}

	os.Setenv("ENVOY_CONCURRENCY", "")

	assert.True(t, foundFlag)
	assert.True(t, foundValue)
}
