package util_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/pkg/errors"

	"github.com/datawire/apro/lib/util"
)

func TestPanicToError(t *testing.T) {
	checkErr := func(t *testing.T, err error) {
		if err == nil {
			t.Error("error: err is nil")
			return
		}

		var k, v string

		////////////////////////////////////////////////////////////////
		k = "err.Error()"
		v = err.Error()
		t.Logf("debug: %s: %q", k, v)
		if !strings.HasPrefix(v, "PANIC: ") {
			t.Errorf("error: %s doesn't look like a panic: %q", k, v)
		}
		if strings.Count(v, "PANIC") != 1 {
			t.Errorf("error: %s looks like it nested wrong: %q", k, v)
		}
		////////////////////////////////////////////////////////////////
		k = "fmt.Sprintf(\"%+v\", err)"
		v = fmt.Sprintf("%+v", err)
		t.Logf("debug: %s: %q", k, v)
		if !strings.HasPrefix(v, "PANIC: ") {
			t.Errorf("error: %s doesn't look like a panic: %q", k, v)
		}
		if strings.Count(v, "PANIC") != 1 {
			t.Errorf("error: %s looks like it nested wrong: %q", k, v)
		}
		if !strings.Contains(v, "panic_test.go") {
			t.Errorf("error: %s doesn't include a stack trace: %q", k, v)
		}
		////////////////////////////////////////////////////////////////
	}
	t.Run("nil", func(t *testing.T) {
		if util.PanicToError(nil) != nil {
			t.Error("error: PanicToError(nil) should be nil")
		}
	})
	t.Run("non-error", func(t *testing.T) { checkErr(t, util.PanicToError("foo")) })
	t.Run("plain-error", func(t *testing.T) { checkErr(t, util.PanicToError(errors.New("err"))) })
	t.Run("wrapped-error", func(t *testing.T) {
		root := fmt.Errorf("x")
		err := util.PanicToError(errors.Wrap(root, "wrapped"))
		checkErr(t, err)
		if errors.Cause(err) != root {
			t.Error("error: error has the wrong cause")
		}
	})
	t.Run("sigsegv", func(t *testing.T) {
		defer func() {
			checkErr(t, util.PanicToError(recover()))
		}()
		var str *string
		fmt.Println(*str) // this will panic
	})
}
