package dlog

import (
	"context"
	"log"
)

type loggerContextKey struct{}

func getLogger(ctx context.Context) Logger {
	logger := ctx.Value(loggerContextKey{})
	if logger == nil {
		return getFallbackLogger()
	}
	return logger.(Logger)
}

// WithLogger returns a copy of ctx with logger associated with it,
// for future calls to {Trace,Debug,Info,Print,Warn,Error}{f,ln,}()
// and StdLogger().
//
// You should only really ever call WithLogger from the initial
// process set up (i.e. directly inside your 'main()' function).
func WithLogger(ctx context.Context, logger Logger) context.Context {
	return context.WithValue(ctx, loggerContextKey{}, logger)
}

// WithField returns a copy of ctx with the logger field key=value
// associated with it, for future calls to
// {Trace,Debug,Info,Print,Warn,Error}{f,ln,}() and StdLogger().
func WithField(ctx context.Context, key string, value interface{}) context.Context {
	return WithLogger(ctx, getLogger(ctx).WithField(key, value))
}

// StdLogger returns a stdlib *log.Logger that uses the Logger
// associated with ctx and logs at the specified loglevel.
//
// Avoid using this functions if at all possible; prefer to use the
// {Trace,Debug,Info,Print,Warn,Error}{f,ln,}() functions.
func StdLogger(ctx context.Context, level LogLevel) *log.Logger {
	return getLogger(ctx).StdLogger(level)
}

func Tracef(c context.Context, f string, a ...interface{}) {
	l := getLogger(c)
	l.Helper()
	l.Tracef(f, a...)
}
func Debugf(c context.Context, f string, a ...interface{}) {
	l := getLogger(c)
	l.Helper()
	l.Debugf(f, a...)
}
func Infof(c context.Context, f string, a ...interface{}) {
	l := getLogger(c)
	l.Helper()
	l.Infof(f, a...)
}
func Printf(c context.Context, f string, a ...interface{}) {
	l := getLogger(c)
	l.Helper()
	l.Printf(f, a...)
}
func Warnf(c context.Context, f string, a ...interface{}) {
	l := getLogger(c)
	l.Helper()
	l.Warnf(f, a...)
}
func Warningf(c context.Context, f string, a ...interface{}) {
	l := getLogger(c)
	l.Helper()
	l.Warningf(f, a...)
}
func Errorf(c context.Context, f string, a ...interface{}) {
	l := getLogger(c)
	l.Helper()
	l.Errorf(f, a...)
}

func Trace(c context.Context, a ...interface{})   { l := getLogger(c); l.Helper(); l.Trace(a...) }
func Debug(c context.Context, a ...interface{})   { l := getLogger(c); l.Helper(); l.Debug(a...) }
func Info(c context.Context, a ...interface{})    { l := getLogger(c); l.Helper(); l.Info(a...) }
func Print(c context.Context, a ...interface{})   { l := getLogger(c); l.Helper(); l.Print(a...) }
func Warn(c context.Context, a ...interface{})    { l := getLogger(c); l.Helper(); l.Warn(a...) }
func Warning(c context.Context, a ...interface{}) { l := getLogger(c); l.Helper(); l.Warning(a...) }
func Error(c context.Context, a ...interface{})   { l := getLogger(c); l.Helper(); l.Error(a...) }

func Traceln(c context.Context, a ...interface{})   { l := getLogger(c); l.Helper(); l.Traceln(a...) }
func Debugln(c context.Context, a ...interface{})   { l := getLogger(c); l.Helper(); l.Debugln(a...) }
func Infoln(c context.Context, a ...interface{})    { l := getLogger(c); l.Helper(); l.Infoln(a...) }
func Println(c context.Context, a ...interface{})   { l := getLogger(c); l.Helper(); l.Println(a...) }
func Warnln(c context.Context, a ...interface{})    { l := getLogger(c); l.Helper(); l.Warnln(a...) }
func Warningln(c context.Context, a ...interface{}) { l := getLogger(c); l.Helper(); l.Warningln(a...) }
func Errorln(c context.Context, a ...interface{})   { l := getLogger(c); l.Helper(); l.Errorln(a...) }
