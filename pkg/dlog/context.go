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

func Tracef(ctx context.Context, f string, a ...interface{})   { getLogger(ctx).Tracef(f, a...) }
func Debugf(ctx context.Context, f string, a ...interface{})   { getLogger(ctx).Debugf(f, a...) }
func Infof(ctx context.Context, f string, a ...interface{})    { getLogger(ctx).Infof(f, a...) }
func Printf(ctx context.Context, f string, a ...interface{})   { getLogger(ctx).Printf(f, a...) }
func Warnf(ctx context.Context, f string, a ...interface{})    { getLogger(ctx).Warnf(f, a...) }
func Warningf(ctx context.Context, f string, a ...interface{}) { getLogger(ctx).Warningf(f, a...) }
func Errorf(ctx context.Context, f string, a ...interface{})   { getLogger(ctx).Errorf(f, a...) }

func Trace(ctx context.Context, a ...interface{})   { getLogger(ctx).Trace(a...) }
func Debug(ctx context.Context, a ...interface{})   { getLogger(ctx).Debug(a...) }
func Info(ctx context.Context, a ...interface{})    { getLogger(ctx).Info(a...) }
func Print(ctx context.Context, a ...interface{})   { getLogger(ctx).Print(a...) }
func Warn(ctx context.Context, a ...interface{})    { getLogger(ctx).Warn(a...) }
func Warning(ctx context.Context, a ...interface{}) { getLogger(ctx).Warning(a...) }
func Error(ctx context.Context, a ...interface{})   { getLogger(ctx).Error(a...) }

func Traceln(ctx context.Context, a ...interface{})   { getLogger(ctx).Traceln(a...) }
func Debugln(ctx context.Context, a ...interface{})   { getLogger(ctx).Debugln(a...) }
func Infoln(ctx context.Context, a ...interface{})    { getLogger(ctx).Infoln(a...) }
func Println(ctx context.Context, a ...interface{})   { getLogger(ctx).Println(a...) }
func Warnln(ctx context.Context, a ...interface{})    { getLogger(ctx).Warnln(a...) }
func Warningln(ctx context.Context, a ...interface{}) { getLogger(ctx).Warningln(a...) }
func Errorln(ctx context.Context, a ...interface{})   { getLogger(ctx).Errorln(a...) }
