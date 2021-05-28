// Copyright 2015-2016 Luke Shumaker
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

//go:generate make

package sd_daemon

import (
	"bytes"
	"io"
	"log/syslog"
	"os"
	"strings"
	"sync"
)

// A Logger writes "<N>"-prefixed lines to an io.Writer, where N is a
// syslog priority number.  It implements mostly the same interface as
// syslog.Writer.
//
// You probably don't need any instance of this other than the
// constant "Log", which uses os.Stderr as the writer.  It is
// implemented as a struct rather than a set of functions so that it
// can be passed around as an implementation of an interface.
//
// Each logging operation makes a single call to the underlying
// Writer's Write method.  A single logging operation may be multiple
// lines; each line will have the prefix, and they will all be written
// in the same Write.  A Logger can be used simultaneously from
// multiple goroutines; it guarantees to serialize access to the
// Writer.
type Logger struct {
	mu  sync.Mutex
	out io.Writer
	buf []byte
}

// NewLogger creates a new Logger.
func NewLogger(w io.Writer) *Logger {
	return &Logger{out: w}
}

// Log is a Logger whose interface is very similar to syslog.Writer,
// but writes to os.Stderr under the assumption that stderr is
// forwarded to syslogd (or journald).
var Log = NewLogger(os.Stderr)

// Cheap version of
//
//     *buf = append(*buf, fmt.Sprintf("<%d>", n)...)
func appendPrefix(buf []byte, n syslog.Priority) []byte {
	var b [21]byte // 21 = (floor(log_10(2^63-1))+1) + len("<>")
	b[len(b)-1] = '>'
	i := len(b) - 2
	for n >= 10 {
		b[i] = byte('0' + n%10)
		n = n / 10
		i--
	}
	b[i] = byte('0' + n)
	i--
	b[i] = '<'
	return append(buf, b[i:]...)
}

// LogString writes a message with the specified priority to the
// log.
func (l *Logger) LogString(pri syslog.Priority, msg string) (n int, err error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	msg = strings.TrimSuffix(msg, "\n")

	// The following is a cheap version of:
	//
	//    prefix := fmt.Sprintf("<%d>", pri)
	//    buf := prefix + strings.Replace(msg, "\n", "\n"+prefix, -1)
	//    return io.WriteString(l.out, buf)

	l.buf = l.buf[:0]
	l.buf = appendPrefix(l.buf, pri) // possible allocation
	prefix := l.buf
	nlines := strings.Count(msg, "\n") + 1
	n = len(msg) + len(prefix)*nlines + 1
	if cap(l.buf) < n {
		l.buf = make([]byte, len(l.buf), n) // allocation
		copy(l.buf, prefix)
	}

	for nlines > 1 {
		nl := strings.IndexByte(msg, '\n')
		l.buf = append(l.buf, msg[:nl+1]...)
		l.buf = append(l.buf, prefix...)
		msg = msg[nl+1:]
		nlines--
	}
	l.buf = append(l.buf, msg...)
	l.buf = append(l.buf, '\n')

	return l.out.Write(l.buf)
}

// LogBytes writes a message with the specified priority to the
// log.
func (l *Logger) LogBytes(pri syslog.Priority, msg []byte) (n int, err error) {
	// Copy/pasted from LogString and
	//  * `strings.` -> `bytes.`
	//  * `"\n"` -> `[]byte{'\n'}`
	l.mu.Lock()
	defer l.mu.Unlock()

	msg = bytes.TrimSuffix(msg, []byte{'\n'})

	l.buf = l.buf[:0]
	l.buf = appendPrefix(l.buf, pri) // possible allocation
	prefix := l.buf
	nlines := bytes.Count(msg, []byte{'\n'}) + 1
	n = len(msg) + len(prefix)*nlines + 1
	if cap(l.buf) < n {
		l.buf = make([]byte, len(l.buf), n) // allocation
		copy(l.buf, prefix)
	}

	for nlines > 1 {
		nl := bytes.IndexByte(msg, '\n')
		l.buf = append(l.buf, msg[:nl+1]...)
		l.buf = append(l.buf, prefix...)
		msg = msg[nl+1:]
		nlines--
	}
	l.buf = append(l.buf, msg...)
	l.buf = append(l.buf, '\n')

	return l.out.Write(l.buf)
}

// Write writes a message with priority syslog.LOG_INFO to the log;
// implementing io.Writer.
func (l *Logger) Write(msg []byte) (n int, err error) {
	n, err = l.LogBytes(syslog.LOG_INFO, msg)
	return n, err
}

// WriteString writes a message with priority syslog.LOG_INFO to the
// log; implementing io.WriteString's interface.
func (l *Logger) WriteString(msg string) (n int, err error) {
	n, err = l.LogString(syslog.LOG_INFO, msg)
	return n, err
}

type loggerWriter struct {
	log *Logger
	pri syslog.Priority
}

func (lw loggerWriter) Write(p []byte) (n int, err error) {
	return lw.log.LogBytes(lw.pri, p)
}

func (lw loggerWriter) WriteString(p string) (n int, err error) {
	return lw.log.LogString(lw.pri, p)
}

// Writer returns an io.Writer that writes messages with the specified
// priority to the log.
func (l *Logger) Writer(pri syslog.Priority) io.Writer {
	return loggerWriter{log: l, pri: pri}
}
