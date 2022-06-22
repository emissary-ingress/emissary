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

func TestGetHealthCheckIPNetworkFamily(t *testing.T) {
	type testCase struct {
		description string
		inputFamily string
		expected    string
	}

	testcases := []testCase{
		{
			description: "non-supported value",
			inputFamily: "not-a-good-value",
			expected:    "tcp",
		},
		{
			description: "ipv6 only",
			inputFamily: "IPV6_ONLY",
			expected:    "tcp6",
		},
		{
			description: "ipv4 only",
			inputFamily: "IPV4_ONLY",
			expected:    "tcp4",
		},
		{
			description: "case-insensitve",
			inputFamily: "ipv4_oNly",
			expected:    "tcp4",
		},
		{
			description: "env var not set",
			inputFamily: "",
			expected:    "tcp",
		},
	}

	for _, tc := range testcases {
		t.Run(tc.description, func(t *testing.T) {
			if tc.inputFamily != "" {
				t.Setenv("AMBASSADOR_HEALTHCHECK_IP_FAMILY", tc.inputFamily)
			}

			result := getHealthCheckIPNetworkFamily()
			assert.Equal(t, tc.expected, result)
		})
	}
}
