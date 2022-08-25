//go:generate make

package dlog

import (
	"context"
	"fmt"
	"log"
)

type loggerContextKey struct{}

// getLogger returns the logger associated with the Context, or else the fallback logger.
//
// You may be asking "Why isn't this exported?  In some cases there might be debug or trace logging
// in loops that where it's important to keep the performance impact as low as possible.  As things
// stand, each log statement will perform a Context lookup!"
//
// The reason is: Exporting it introduces the possibility of misuse, and so at this point exporting
// it "for performance" would be premature optimization.  If we ever do see this causing a
// performance problem, then we can export it.  But until then, let's make it hard to misuse.
//
// You see, it was exported back in the days before https://github.com/datawire/apro/pull/1818 (in
// fact dlog.GetLogger(ctx).Infoln(…) was the only way to do it for a long time).  What we saw with
// that was that it's really easy to end up calling `logger = logger.WithField(…)` and ctx =
// `dlog.WithField(ctx, …)` separately and having the separate logger and the ctx drift from
// eachother (often, you'll do the former, not updating the ctx, then later someone passes the ctx
// to another function, so that function's logger doesn't have the updates).  This is a misuse.
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
//
// If the logger implements OptimizedLogger, then dlog will take
// advantage of that.
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
// Avoid using this function if at all possible; prefer to use the
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
	if opt, ok := l.(OptimizedLogger); ok {
		opt.UnformattedLog(lvl, args...)
	} else {
		l.Log(lvl, fmt.Sprint(args...))
	}
}

func Logln(ctx context.Context, lvl LogLevel, args ...interface{}) {
	l := getLogger(ctx)
	l.Helper()
	if opt, ok := l.(OptimizedLogger); ok {
		opt.UnformattedLogln(lvl, args...)
	} else {
		l.Log(lvl, sprintln(args...))
	}
}

func Logf(ctx context.Context, lvl LogLevel, format string, args ...interface{}) {
	l := getLogger(ctx)
	l.Helper()
	if opt, ok := l.(OptimizedLogger); ok {
		opt.UnformattedLogf(lvl, format, args...)
	} else {
		l.Log(lvl, fmt.Sprintf(format, args...))
	}
}
