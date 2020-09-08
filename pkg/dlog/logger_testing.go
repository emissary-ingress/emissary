package dlog

import (
	"fmt"
	"log"
	"sort"
	"strings"
	"testing"

	"github.com/pkg/errors"
)

type tbWrapper struct {
	testing.TB
	failOnError bool
	fields      map[string]interface{}
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

func (w tbWrapper) log(level LogLevel, msg string) {
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

func (w tbWrapper) Log(level LogLevel, a ...interface{}) {
	w.Helper()
	w.log(level, fmt.Sprint(a...))
}

func (w tbWrapper) Logln(level LogLevel, a ...interface{}) {
	w.Helper()
	w.log(level, fmt.Sprintln(a...))
}

func (w tbWrapper) Logf(level LogLevel, format string, a ...interface{}) {
	w.Helper()
	w.log(level, fmt.Sprintf(format, a...))
}

func (w tbWrapper) Tracef(f string, a ...interface{})   { w.Helper(); w.Logf(LogLevelTrace, f, a...) }
func (w tbWrapper) Debugf(f string, a ...interface{})   { w.Helper(); w.Logf(LogLevelDebug, f, a...) }
func (w tbWrapper) Infof(f string, a ...interface{})    { w.Helper(); w.Logf(LogLevelInfo, f, a...) }
func (w tbWrapper) Printf(f string, a ...interface{})   { w.Helper(); w.Logf(LogLevelInfo, f, a...) }
func (w tbWrapper) Warnf(f string, a ...interface{})    { w.Helper(); w.Logf(LogLevelWarn, f, a...) }
func (w tbWrapper) Warningf(f string, a ...interface{}) { w.Helper(); w.Logf(LogLevelWarn, f, a...) }
func (w tbWrapper) Errorf(f string, a ...interface{})   { w.Helper(); w.Logf(LogLevelError, f, a...) }

func (w tbWrapper) Trace(a ...interface{})   { w.Helper(); w.Log(LogLevelTrace, a...) }
func (w tbWrapper) Debug(a ...interface{})   { w.Helper(); w.Log(LogLevelDebug, a...) }
func (w tbWrapper) Info(a ...interface{})    { w.Helper(); w.Log(LogLevelInfo, a...) }
func (w tbWrapper) Print(a ...interface{})   { w.Helper(); w.Log(LogLevelInfo, a...) }
func (w tbWrapper) Warn(a ...interface{})    { w.Helper(); w.Log(LogLevelWarn, a...) }
func (w tbWrapper) Warning(a ...interface{}) { w.Helper(); w.Log(LogLevelWarn, a...) }
func (w tbWrapper) Error(a ...interface{})   { w.Helper(); w.Log(LogLevelError, a...) }

func (w tbWrapper) Traceln(a ...interface{})   { w.Helper(); w.Logln(LogLevelTrace, a...) }
func (w tbWrapper) Debugln(a ...interface{})   { w.Helper(); w.Logln(LogLevelDebug, a...) }
func (w tbWrapper) Infoln(a ...interface{})    { w.Helper(); w.Logln(LogLevelInfo, a...) }
func (w tbWrapper) Println(a ...interface{})   { w.Helper(); w.Logln(LogLevelInfo, a...) }
func (w tbWrapper) Warnln(a ...interface{})    { w.Helper(); w.Logln(LogLevelWarn, a...) }
func (w tbWrapper) Warningln(a ...interface{}) { w.Helper(); w.Logln(LogLevelWarn, a...) }
func (w tbWrapper) Errorln(a ...interface{})   { w.Helper(); w.Logln(LogLevelError, a...) }

// WrapTB converts a testing.TB (that is: either a *testing.T or a
// *testing.B) into a generic Logger.
//
// Naturally, you should only use this from inside of your *_test.go
// files.
func WrapTB(in testing.TB, failOnError bool) Logger {
	return tbWrapper{
		TB:          in,
		failOnError: failOnError,
		fields:      map[string]interface{}{},
	}
}

type tbWriter struct {
	w tbWrapper
	l LogLevel
}

func (w tbWriter) Write(data []byte) (n int, err error) {
	w.w.Helper()
	w.w.log(w.l, string(data))
	return len(data), nil
}

func (w tbWrapper) StdLogger(l LogLevel) *log.Logger {
	return log.New(tbWriter{w, l}, "", 0)
}
