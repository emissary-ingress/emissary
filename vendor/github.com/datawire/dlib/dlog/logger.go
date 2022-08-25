// Package dlog implements a generic logger facade.
//
// There are three first-class things of value in this package:
//
// First: The Logger interface.  This is a simple structured logging
// interface that is mostly trivial to implement on top of most
// logging backends, and allows library code to not need to care about
// what specific logging system the calling program uses.
//
// Second: The WithLogger and WithField functions for affecting the
// logging for a context.
//
// Third: The actual logging functions.  If you are writing library
// code and want to log things, then you should take a context.Context
// as an argument, and then call dlog.{Level}{,f,ln}(ctx, args) to
// log.
package dlog

import (
	"log"
)

// Logger is a generic logging interface that most loggers implement,
// so that consumers don't need to care about the actual log
// implementation.
//
// Note that unlike logrus.FieldLogger, it does not include Fatal or
// Panic logging options.  Do proper error handling!  Return those
// errors!
//
// Because this interface cannot change without breaking backward
// compatibility, it is frozen and cannot be changed.
type Logger interface {
	// Helper marks the calling function as a logging helper
	// function.  This way loggers can report the line that the
	// log came from, while excluding functions that are part of
	// dlog itself.
	//
	// See also: testing.T.Helper
	Helper()

	// WithField returns a copy of the logger with the
	// structured-logging field key=value associated with it, for
	// future calls to .Log().
	WithField(key string, value interface{}) Logger

	// StdLogger returns a stdlib *log.Logger that writes to this
	// Logger at the specified loglevel; for use with external
	// libraries that demand a stdlib *log.Logger.  Since
	StdLogger(LogLevel) *log.Logger

	// Log actually logs a message.
	Log(level LogLevel, msg string)
}

// OptimizedLogger is a Logger that takes on the complexity of
// formatting log lines in to a string.  Normally this is done by dlog
// itself before passing the formatted string to the Logger.  However,
// if the Logger is able to do these things itself then it might save
// some time by not actually formatting messages that won't be
// printed.
//
// Because OptimizedLogger is an opt-in optimization that does not
// affect correctness, its definition may change between dlib
// versions.
type OptimizedLogger interface {
	Logger

	// UnformattedLog formats then logs a message.  The message is
	// formatted using the default formats for its operands and
	// adds spaces between operands when neither is a string; in
	// the manner of fmt.Print().
	UnformattedLog(level LogLevel, args ...interface{})

	// UnformattedLogln formats then logs a message.  The message
	// is formatted using the default formats for its operands and
	// always adds spaces between operands; in the manner of
	// fmt.Println() but without appending a newline.
	UnformattedLogln(level LogLevel, args ...interface{})

	// UnformattedLogf formats then logs a message.  The message is
	// formatted according to the format specifier; in the manner
	// of fmt.Printf().
	UnformattedLogf(level LogLevel, format string, args ...interface{})
}

// LogLevel is an abstracted common log-level type for Logger.StdLogger().
type LogLevel uint32

const (
	// LogLevelError is for errors that should definitely be noted.
	LogLevelError LogLevel = iota
	// LogLevelWarn is for non-critical entries that deserve eyes.
	LogLevelWarn
	// LogLevelInfo is for general operational entries about what's
	// going on inside the application.
	LogLevelInfo
	// LogLevelDebug is for debugging.  Very verbose logging.
	LogLevelDebug
	// LogLevelTrace is for extreme debugging.  Even finer-grained
	// informational events than the Debug.
	LogLevelTrace
)
