package acp_test

import (
	"testing"

	"github.com/datawire/ambassador/v2/pkg/acp"
)

func check(t *testing.T, hostport string, wanted bool) {
	if acp.HostPortIsLocal(hostport) != wanted {
		t.Errorf("HostPort %s: wanted %v, got %v", hostport, wanted, acp.HostPortIsLocal(hostport))
	}
}

func TestHostIsLocal(t *testing.T) {
	// HostIsLocal requires port numbers.
	check(t, "localhost", false)
	check(t, "127.0.0.1", false)
	check(t, "LOCALHOST", false)
	check(t, "127.0.0.2", false)
	check(t, "LoCalHosT", false)
	check(t, "localhostt", false)
	check(t, "localhop", false)
	check(t, "::1", false)
	check(t, "::2", false)

	check(t, "localhost:9999", true)
	check(t, "127.0.0.1:9999", true)
	check(t, "LOCALHOST:9999", false)
	check(t, "127.0.0.2:9999", false)
	check(t, "LoCalHosT:9999", false)
	check(t, "localhostt:9999", false)
	check(t, "localhop:9999", false)
	check(t, "[::1]:9999", true)
	check(t, "[::2]:9999", false)

	// I'm not sure if the square braces are valid without a port number,
	// but HostPortIsLocal doesn't do them either.
	check(t, "[::1]", false)
	check(t, "[::2]", false)
}
