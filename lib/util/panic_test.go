package util

import (
	"fmt"
	"strings"
	"testing"

	"github.com/pkg/errors"
)

func TestPanicToError(t *testing.T) {
	checkErr := func(err error) {
		if err == nil {
			t.Error("error is nil")
		}
		if !strings.HasPrefix(err.Error(), "PANIC: ") {
			t.Error("error doesn't look like a panic")
		}
		if strings.Count(err.Error(), "PANIC") != 1 {
			t.Error("error looks like it nested wrong")
		}
		if !strings.Contains(fmt.Sprintf("%+v", err), "panic_test.go") {
			t.Error("error doesn't include a stack track with +v")
		}
	}
	if PanicToError(nil) != nil {
		t.Error("PanicToError(nil) should be nil")
	}
	checkErr(PanicToError("foo"))
	checkErr(PanicToError(errors.New("err")))
	root := fmt.Errorf("x")
	err := PanicToError(errors.Wrap(root, "wrapped"))
	checkErr(err)
	if errors.Cause(err) != root {
		t.Error("error has the wrong cause")
	}
}
