package testutil

import (
	"testing"
)

// Assert ..
type Assert struct {
	T *testing.T
}

// StrEQ ..
func (a *Assert) StrEQ(e string, c string) {
	if e != c {
		a.T.Fatalf("Expected '%s' got '%s'", e, c)
	}
}

// IntEQ ..
func (a *Assert) IntEQ(e int, c int) {
	if e != c {
		a.T.Fatalf("Expected '%v' got '%v'", e, c)
	}
}

// NotNil ..
func (a *Assert) NotNil(c interface{}) {
	if c == nil {
		a.T.Fatalf("Expected not NIL got '%v'", c)
	}
}
