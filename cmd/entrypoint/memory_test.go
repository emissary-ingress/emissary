package entrypoint

import "testing"

// This doesn't actually assert anything (yet), but it's still useful for iterating on the
// implementation.
func TestPrintMemoryUsage(t *testing.T) {
	for _, mu := range GetAllMemoryUsage() {
		t.Log(mu.String())
	}
}
