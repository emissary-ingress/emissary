package testutil

import (
	"net/http"
	"net/http/httputil"
	"testing"
)

// Assert has convenient functions for doing test assertions.
type Assert struct {
	T *testing.T
}

// StrEQ asserts that two strings are equivalent.
func (a *Assert) StrEQ(e string, c string) {
	a.T.Helper()
	if e != c {
		a.T.Fatalf("Expected '%s' got '%s'", e, c)
	}
}

// StrEQ asserts that two strings are not equivalent.
func (a *Assert) StrNotEQ(e string, c string) {
	a.T.Helper()
	if e == c {
		a.T.Fatalf("Expected '%s' got '%s'", e, c)
	}
}

// IntEQ assert that two integers are the same.
func (a *Assert) IntEQ(e int, c int) {
	a.T.Helper()
	if e != c {
		a.T.Fatalf("Expected '%v' got '%v'", e, c)
	}
}

// NotNil asserts that the object is not nil.
func (a *Assert) NotNil(c interface{}) {
	a.T.Helper()
	if c == nil {
		a.T.Fatalf("Expected not NIL got '%v'", c)
	}
}

// Nil asserts that the object is nil.
func (a *Assert) Nil(c interface{}) {
	a.T.Helper()
	if c != nil {
		a.T.Fatalf("Expected NIL got '%v'", c)
	}
}

// StrNotEmpty asserts that string is not empty.
func (a *Assert) StrNotEmpty(e string) {
	a.T.Helper()
	if len(e) == 0 {
		a.T.Fatalf("Expected not empty string got empty")
	}
}

func (a *Assert) NotError(err error) {
	a.T.Helper()
	if err != nil {
		a.T.Fatalf("Unexpected error %v", err)
	}
}

func (a *Assert) HTTPResponseStatusEQ(r *http.Response, expected int) {
	a.T.Helper()
	if r.StatusCode != expected {
		data, _ := httputil.DumpResponse(r, true)
		a.T.Fatalf("Unexpected HTTP response status <%d> wanted <%d>\n\n%s", r.StatusCode, expected, data)
	}
}
