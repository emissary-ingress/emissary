package dlog

import (
	"io"
	"log"
	"runtime"
	"strings"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

type logrusLogger interface {
	WithField(key string, value interface{}) *logrus.Entry
	WriterLevel(level logrus.Level) *io.PipeWriter
	Log(level logrus.Level, args ...interface{})
	Logln(level logrus.Level, args ...interface{})
	Logf(level logrus.Level, format string, args ...interface{})
}

type logrusWrapper struct {
	logrusLogger
}

// Helper does nothing--we use a Logrus Hook instead (see below).
func (l logrusWrapper) Helper() {}

func (l logrusWrapper) WithField(key string, value interface{}) Logger {
	return logrusWrapper{l.logrusLogger.WithField(key, value)}
}

var dlogLevel2logrusLevel = map[LogLevel]logrus.Level{
	LogLevelError: logrus.ErrorLevel,
	LogLevelWarn:  logrus.WarnLevel,
	LogLevelInfo:  logrus.InfoLevel,
	LogLevelDebug: logrus.DebugLevel,
	LogLevelTrace: logrus.TraceLevel,
}

func (l logrusWrapper) StdLogger(level LogLevel) *log.Logger {
	logrusLevel, ok := dlogLevel2logrusLevel[level]
	if !ok {
		panic(errors.Errorf("invalid LogLevel: %d", level))
	}
	return log.New(l.logrusLogger.WriterLevel(logrusLevel), "", 0)
}

func (l logrusWrapper) Log(level LogLevel, args ...interface{}) {
	logrusLevel, ok := dlogLevel2logrusLevel[level]
	if !ok {
		panic(errors.Errorf("invalid LogLevel: %d", level))
	}
	l.logrusLogger.Log(logrusLevel, args...)
}

func (l logrusWrapper) Logf(level LogLevel, format string, args ...interface{}) {
	logrusLevel, ok := dlogLevel2logrusLevel[level]
	if !ok {
		panic(errors.Errorf("invalid LogLevel: %d", level))
	}
	l.logrusLogger.Logf(logrusLevel, format, args...)
}

func (l logrusWrapper) Logln(level LogLevel, args ...interface{}) {
	logrusLevel, ok := dlogLevel2logrusLevel[level]
	if !ok {
		panic(errors.Errorf("invalid LogLevel: %d", level))
	}
	l.logrusLogger.Logln(logrusLevel, args...)
}

// WrapLogrus converts a logrus *Logger into a generic Logger.
//
// You should only really ever call WrapLogrus from the initial
// process set up (i.e. directly inside your 'main()' function), and
// you should pass the result directly to WithLogger.
func WrapLogrus(in *logrus.Logger) Logger {
	in.AddHook(logrusFixCallerHook{})
	return logrusWrapper{in}
}

type logrusFixCallerHook struct{}

func (logrusFixCallerHook) Levels() []logrus.Level {
	return logrus.AllLevels
}

func (logrusFixCallerHook) Fire(entry *logrus.Entry) error {
	if entry.Caller != nil && strings.HasPrefix(entry.Caller.Function, dlogPackage+".") {
		entry.Caller = getCaller()
	}
	return nil
}

const (
	dlogPackage            = "github.com/datawire/dlib/dlog"
	logrusPackage          = "github.com/sirupsen/logrus"
	maximumCallerDepth int = 25
	minimumCallerDepth int = 2 // runtime.Callers + getCaller
)

// Duplicate of logrus.getCaller() because Logrus doesn't have the
// kind if skip/.Helper() functionality that testing.TB has.
//
// https://github.com/sirupsen/logrus/issues/972
func getCaller() *runtime.Frame {
	// Restrict the lookback frames to avoid runaway lookups
	pcs := make([]uintptr, maximumCallerDepth)
	depth := runtime.Callers(minimumCallerDepth, pcs)
	frames := runtime.CallersFrames(pcs[:depth])

	for f, again := frames.Next(); again; f, again = frames.Next() {
		// If the caller isn't part of this package, we're done
		if strings.HasPrefix(f.Function, logrusPackage+".") {
			continue
		}
		if strings.HasPrefix(f.Function, dlogPackage+".") {
			continue
		}
		return &f //nolint:scopelint
	}

	// if we got here, we failed to find the caller's context
	return nil
}
