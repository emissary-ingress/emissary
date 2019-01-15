package testutil

import (
	"testing"
)

// Assert has convenient functions for doing test assertions.
type Assert struct {
	T *testing.T
}

// StrEQ asserts that two strings are equivalent.
func (a *Assert) StrEQ(e string, c string) {
	if e != c {
		a.T.Fatalf("Expected '%s' got '%s'", e, c)
	}
}

// IntEQ assert that two integers are the same.
func (a *Assert) IntEQ(e int, c int) {
	if e != c {
		a.T.Fatalf("Expected '%v' got '%v'", e, c)
	}
}

// NotNil asserts that the object is not nil.
func (a *Assert) NotNil(c interface{}) {
	if c == nil {
		a.T.Fatalf("Expected not NIL got '%v'", c)
	}
}

// Nil asserts that the object is nil.
func (a *Assert) Nil(c interface{}) {
	if c != nil {
		a.T.Fatalf("Expected NIL got '%v'", c)
	}
}

// StrNotEmpty asserts that string is not empty.
func (a *Assert) StrNotEmpty(e string) {
	if len(e) == 0 {
		a.T.Fatalf("Expected not empty string got empty")
	}
}
