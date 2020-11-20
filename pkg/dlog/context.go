//go:generate make

package dlog

import (
	"context"
	"fmt"
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
// {Trace,Debug,Info,Print,Warn,Error}{f,ln,}() functions.  You should
// only use this for working with external libraries that demand a
// stdlib *log.Logger.
func StdLogger(ctx context.Context, level LogLevel) *log.Logger {
	return getLogger(ctx).StdLogger(level)
}

func sprintln(args ...interface{}) string {
	// Trim the trailing newline; what we care about is that spaces are added in between
	// arguments, not that there's a trailing newline.  See also: logrus.Entry.sprintlnn
	msg := fmt.Sprintln(args...)
	return msg[:len(msg)-1]
}

// If you change any of these, you should also change convenience.go.gen and run `make generate`.

func Log(ctx context.Context, lvl LogLevel, args ...interface{}) {
	l := getLogger(ctx)
	l.Helper()
	l.Log(lvl, fmt.Sprint(args...))
}

func Logln(ctx context.Context, lvl LogLevel, args ...interface{}) {
	l := getLogger(ctx)
	l.Helper()
	l.Log(lvl, sprintln(args...))
}

func Logf(ctx context.Context, lvl LogLevel, format string, args ...interface{}) {
	l := getLogger(ctx)
	l.Helper()
	l.Log(lvl, fmt.Sprintf(format, args...))
}
