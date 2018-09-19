package testutil

import (
	"testing"
)

// Assert TODO(gsagula): comment
type Assert struct {
	T *testing.T
}

// StrEQ TODO(gsagula): comment
func (a *Assert) StrEQ(e string, c string) {
	if e != c {
		a.T.Fatalf("Expected '%s' got '%s'", e, c)
	}
}

// IntEQ TODO(gsagula): comment
func (a *Assert) IntEQ(e int, c int) {
	if e != c {
		a.T.Fatalf("Expected '%v' got '%v'", e, c)
	}
}

// NotNil TODO(gsagula): comment
func (a *Assert) NotNil(c interface{}) {
	if c == nil {
		a.T.Fatalf("Expected not NIL got '%v'", c)
	}
}
