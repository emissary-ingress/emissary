package testutil

import (
	"math/big"
	"net/http"
	"net/http/httputil"
	"testing"
)

// Assert has convenient functions for doing test assertions.
type Assert struct {
	T                 testing.TB
	SkipInsteadOfFail bool
}

func (a *Assert) fatalf(format string, args ...interface{}) {
	a.T.Helper()
	if a.SkipInsteadOfFail {
		a.T.Logf(format, args...)
		a.T.SkipNow()
	} else {
		a.T.Fatalf(format, args...)
	}
}

func (a *Assert) Bool(b bool) {
	a.T.Helper()
	if !b {
		a.fatalf("Assertion failed")
	}
}

// StrEQ asserts that two strings are equivalent.
func (a *Assert) StrEQ(expected string, received string) {
	a.T.Helper()
	if expected != received {
		a.fatalf(`Assertion failed:
Expected: %q
Received: %q`,
			expected, received)
	}
}

// StrEQ asserts that two strings are not equivalent.
func (a *Assert) StrNotEQ(expected string, received string) {
	a.T.Helper()
	if expected == received {
		a.fatalf(`Assertion failed:
Expected: anything but %q
Received:              %q`,
			expected, received)
	}
}

// IntEQ assert that two integers are the same.
func (a *Assert) IntEQ(expected int, received int) {
	a.T.Helper()
	if expected != received {
		a.fatalf(`Assertion failed:
Expected: %d
Received: %d`,
			expected, received)
	}
}

// IntEQ assert that two integers are the same.
func (a *Assert) BigIntEQ(expected *big.Int, received *big.Int) {
	a.T.Helper()
	if expected.Cmp(received) != 0 {
		a.fatalf(`Assertion failed:
Expected: %v
Received: %v`,
			expected, received)
	}
}

// StrNotEmpty asserts that string is not empty.
func (a *Assert) StrNotEmpty(expected string) {
	a.T.Helper()
	if len(expected) == 0 {
		a.fatalf(`Assertion failed:
Expected: any non-empty string
Received: %q`,
			expected)
	}
}

func (a *Assert) NotError(err error) {
	a.T.Helper()
	if err != nil {
		a.fatalf("Unexpected error: %v", err)
	}
}

func (a *Assert) HTTPResponseStatusEQ(r *http.Response, expected int) {
	a.T.Helper()
	if r.StatusCode != expected {
		data, _ := httputil.DumpResponse(r, true)
		a.fatalf("Unexpected HTTP response status <%d> wanted <%d>\n\n%s", r.StatusCode, expected, data)
	}
}
