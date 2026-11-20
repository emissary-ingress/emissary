package acp

import (
	"testing"
)

// TestEnvoyWatcherReadyURL checks which port the EnvoyWatcher polls for /ready, given
// AMBASSADOR_READY_PORT.
//
// The Envoy /ready listener is configured by python/ambassador/envoy/v3/v3ready.py, which falls back
// to 8006 for any port outside [1, 32767). The EnvoyWatcher has to land on the same port as that
// listener, or it never sees Envoy become ready.
func TestEnvoyWatcherReadyURL(t *testing.T) {
	testcases := map[string]struct {
		env      string // value of AMBASSADOR_READY_PORT; "" means unset
		expected string
	}{
		"unset":                    {env: "", expected: "http://localhost:8006/ready"},
		"custom":                   {env: "8010", expected: "http://localhost:8010/ready"},
		"lowest valid":             {env: "1", expected: "http://localhost:1/ready"},
		"highest valid":            {env: "32766", expected: "http://localhost:32766/ready"},
		"zero":                     {env: "0", expected: "http://localhost:8006/ready"},
		"negative":                 {env: "-1", expected: "http://localhost:8006/ready"},
		"not a number":             {env: "420/1337.abcd", expected: "http://localhost:8006/ready"},
		"first rejected by python": {env: "32767", expected: "http://localhost:8006/ready"},
		"above 15 bits":            {env: "32768", expected: "http://localhost:8006/ready"},
		"ephemeral range":          {env: "40000", expected: "http://localhost:8006/ready"},
		"highest TCP port":         {env: "65535", expected: "http://localhost:8006/ready"},
		"above the TCP port range": {env: "70000", expected: "http://localhost:8006/ready"},
		"overflows 64 bits":        {env: "99999999999999999999999", expected: "http://localhost:8006/ready"},
	}

	for name, tc := range testcases {
		t.Run(name, func(t *testing.T) {
			t.Setenv("AMBASSADOR_READY_PORT", tc.env)

			ew := NewEnvoyWatcher()

			if ew.defaultReadyURL != tc.expected {
				t.Errorf("AMBASSADOR_READY_PORT=%q: polling %q, wanted %q", tc.env, ew.defaultReadyURL, tc.expected)
			}
		})
	}
}
