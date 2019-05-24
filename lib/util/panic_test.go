package util

import (
	"fmt"
	"strings"
	"testing"

	"github.com/pkg/errors"
)

func TestPanicToError(t *testing.T) {
	checkErr := func(t *testing.T, err error) {
		if err == nil {
			t.Error("err is nil")
		}

		plain := err.Error()
		if !strings.HasPrefix(plain, "PANIC: ") {
			t.Errorf("%s doesn't look like a panic: %q", "err.Error()", plain)
		}
		if strings.Count(plain, "PANIC") != 1 {
			t.Errorf("%s looks like it nested wrong: %q", "err.Error()", plain)
		}

		verbose := fmt.Sprintf("%+v", err)
		if !strings.HasPrefix(verbose, "PANIC: ") {
			t.Errorf("%s doesn't look like a panic: %q", "fmt.Sprintf(\"%+v\", err)", verbose)
		}
		if strings.Count(verbose, "PANIC") != 1 {
			t.Errorf("%s looks like it nested wrong: %q", "fmt.Sprintf(\"%+v\", err)", verbose)
		}

		if !strings.Contains(verbose, "panic_test.go") {
			t.Errorf("%s doesn't include a stack track: %q", "fmt.Sprintf(\"%+v\", err)", verbose)
		}
		t.Logf("verbose: %q", verbose)
	}
	t.Run("nil", func(t *testing.T) {
		if PanicToError(nil) != nil {
			t.Error("PanicToError(nil) should be nil")
		}
	})
	t.Run("non-error", func(t *testing.T) { checkErr(t, PanicToError("foo")) })
	t.Run("plain-error", func(t *testing.T) { checkErr(t, PanicToError(errors.New("err"))) })
	t.Run("wrapped-error", func(t *testing.T) {
		root := fmt.Errorf("x")
		err := PanicToError(errors.Wrap(root, "wrapped"))
		checkErr(t, err)
		if errors.Cause(err) != root {
			t.Error("error has the wrong cause")
		}
	})
	t.Run("sigsegv", func(t *testing.T) {
		defer func() {
			checkErr(t, PanicToError(recover()))
		}()
		var str *string
		fmt.Println(*str) // this will panic
	})
}
