package dlog

import (
	"context"
	"fmt"
	"log"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/pkg/errors"
)

type tbWrapper struct {
	testing.TB
	failOnError   bool
	logTimestamps bool
	fields        map[string]interface{}
}

func (w tbWrapper) WithField(key string, value interface{}) Logger {
	ret := tbWrapper{
		TB:     w.TB,
		fields: make(map[string]interface{}, len(w.fields)+1),
	}
	for k, v := range w.fields {
		ret.fields[k] = v
	}
	ret.fields[key] = value
	return ret
}

func (w tbWrapper) Log(level LogLevel, msg string) {
	w.Helper()
	fields := make(map[string]interface{}, len(w.fields)+2)
	for k, v := range w.fields {
		fields[k] = v
	}
	fields["msg"] = msg
	var ok bool
	fields["level"], ok = map[LogLevel]string{
		LogLevelError: "error",
		LogLevelWarn:  "warn",
		LogLevelInfo:  "info",
		LogLevelDebug: "debug",
		LogLevelTrace: "trace",
	}[level]
	if !ok {
		panic(errors.Errorf("invalid LogLevel: %d", level))
	}

	if w.logTimestamps {
		fields["timestamp"] = time.Now().Format("2006-01-02 15:04:05.0000")
	}

	parts := make([]string, 0, len(fields))
	for k := range fields {
		parts = append(parts, k)
	}
	sort.Strings(parts)
	for i, k := range parts {
		parts[i] = fmt.Sprintf("%s=%#v", k, fields[k])
	}
	str := strings.Join(parts, " ")

	switch level {
	case LogLevelError:
		if w.failOnError {
			w.TB.Error(str)
		} else {
			w.TB.Log(str)
		}
	case LogLevelWarn, LogLevelInfo, LogLevelDebug, LogLevelTrace:
		w.TB.Log(str)
	}
}

// WrapTB converts a testing.TB (that is: either a *testing.T or a *testing.B) into a generic
// Logger.
//
// Naturally, you should only use this from inside of your *_test.go files.  The failOnError
// argument controls whether calling any of the dlog.Error{,f,ln} functions should cause the test to
// fail.
//
// This is considered deprecated; you should consider using NewTestContext (which calls this)
// instead.
func WrapTB(in testing.TB, failOnError bool) Logger {
	return wrapTB(in, WithFailOnError(failOnError))
}

func wrapTB(in testing.TB, opts ...TestContextOption) Logger {
	wrapper := tbWrapper{
		TB:            in,
		fields:        map[string]interface{}{},
		logTimestamps: true,
	}
	for _, opt := range opts {
		opt(&wrapper)
	}
	return wrapper
}

type tbWriter struct {
	w tbWrapper
	l LogLevel
}

func (w tbWriter) Write(data []byte) (n int, err error) {
	w.w.Helper()
	w.w.Log(w.l, string(data))
	return len(data), nil
}

func (w tbWrapper) StdLogger(l LogLevel) *log.Logger {
	return log.New(tbWriter{w, l}, "", 0)
}

// TestContextOption represents options that can be set on test contexts
type TestContextOption func(*tbWrapper)

// WithFailOnError sets a test context to fail on calling any of the dlog.Error{,f,ln} functions.
// If not given, defaults to false
func WithFailOnError(failOnError bool) TestContextOption {
	return func(w *tbWrapper) {
		w.failOnError = failOnError
	}
}

// WithTimestampLogging sets a test context to always log timestamps
// Note that these will be logged as log fields.
// If not given, defaults to true
func WithTimestampLogging(logTimestamps bool) TestContextOption {
	return func(w *tbWrapper) {
		w.logTimestamps = logTimestamps
	}
}

// NewTestContext is like NewTestContextWithOpts but allows for the failOnError option to be set
// as a boolean. It is kept for backward-compatibility, new code should prefer NewTestContextWithOpts
func NewTestContext(t testing.TB, failOnError bool) context.Context {
	return NewTestContextWithOpts(t, WithFailOnError(failOnError))
}

// NewTestContextWithOpts takes a testing.TB (that is: either a *testing.T or a *testing.B) and returns a
// good default Context to use in unit test.  The Context will have dlog configured to log using the
// Go test runner's built-in logging facilities.  The context will be canceled when the test
// terminates.  The failOnError argument controls whether calling any of the dlog.Error{,f,ln}
// functions should cause the test to fail.
//
// Naturally, you should only use this from inside of your *_test.go files.
func NewTestContextWithOpts(t testing.TB, opts ...TestContextOption) context.Context {
	ctx := context.Background()
	ctx = WithLogger(ctx, wrapTB(t, opts...))
	ctx, cancel := context.WithCancel(ctx)
	t.Cleanup(cancel)
	return ctx
}
