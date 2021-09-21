package testutils

import (
	"bytes"
	"fmt"
	"path/filepath"
	"runtime"
	"strconv"
	"testing"
	"time"
)

type Retryable struct {
	logger  *bytes.Buffer
	failed  bool
	skipped bool
	abort   bool
	name    string
	t       *testing.T
}

func (r *Retryable) Parallel() {
	r.t.Parallel()
}

func (r *Retryable) Error(v ...interface{}) {
	r.log(v...)
	r.failed = true
}

func (r *Retryable) Fail() {
	r.failed = true
}

func (r *Retryable) FailNow() {
	r.failed = true
	r.abort = true
}

func (r *Retryable) Fatal(v ...interface{}) {
	r.log(v...)
	r.failed = true
	r.abort = true
}

func (r *Retryable) Fatalf(s string, v ...interface{}) {
	r.logf(s, v...)
	r.failed = true
	r.abort = true
}

func (r *Retryable) Helper() {
	r.t.Helper()
}

func (r *Retryable) Errorf(s string, v ...interface{}) {
	r.log(v...)
	r.failed = true
}

func (r *Retryable) Skip(v ...interface{}) {
	r.log(v...)
	r.skipped = true
}

func (r *Retryable) Skipf(s string, v ...interface{}) {
	r.logf(s, v...)
	r.skipped = true
}

func (r *Retryable) SkipNow() {
	r.skipped = true
}

func (r *Retryable) Skipped() bool {
	return r.skipped
}

func (r *Retryable) TempDir() string {
	return r.t.TempDir()
}

func (r *Retryable) Log(v ...interface{}) {
	r.log(v...)
}

func (r *Retryable) Name() string {
	return r.name
}

func (r *Retryable) Cleanup(f func()) {
	f()
}

func (r *Retryable) Failed() bool {
	return r.failed
}

func (r *Retryable) Logf(s string, v ...interface{}) {
	r.logf(s, v...)
}

func (r *Retryable) log(v ...interface{}) {
	fmt.Fprint(r.logger, "\n")
	fmt.Fprint(r.logger, lineNumber())
	fmt.Fprint(r.logger, v...)
}

func (r *Retryable) logf(s string, v ...interface{}) {
	fmt.Fprint(r.logger, "\n")
	fmt.Fprint(r.logger, lineNumber())
	fmt.Fprintf(r.logger, s, v...)
}

func lineNumber() string {
	_, file, line, ok := runtime.Caller(3)
	if !ok {
		return ""
	}
	return filepath.Base(file) + ":" + strconv.Itoa(line) + ": "
}

func Retry(t *testing.T, numRetries int, f func(r *Retryable)) bool {

	var lastLog *bytes.Buffer
	for i := 0; i < numRetries; i++ {
		r := &Retryable{
			logger:  &bytes.Buffer{},
			failed:  false,
			skipped: false,
			name:    t.Name(),
			t:       t,
		}
		f(r)
		if r.skipped {
			t.Skip()
			return true
		}
		if !r.failed {
			return true
		}
		lastLog = r.logger
		if r.abort {
			break
		}
		time.Sleep(time.Second * 5)
	}
	t.Logf("Failed after %d attempts:%s", numRetries, lastLog.String())
	t.Fail()

	return false
}
