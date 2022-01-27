package derror

import (
	"fmt"
	"io"

	"github.com/pkg/errors"
)

// unwrapper is not exported by "errors".
type unwrapper interface {
	Unwrap() error
}

// causer is not exported by "github.com/pkg/errors".
type causer interface {
	Cause() error
}

// stackTracer is not exported by "github.com/pkg/errors".
type stackTracer interface {
	StackTrace() errors.StackTrace
}

// featurefulError documents the features of github.com/pkg/errors.WithStack() and
// github.com/pkg/errors.Errorf().
type featurefulError interface {
	error
	stackTracer
	fmt.Formatter
}

type panicError struct {
	err featurefulError
}

func (pe panicError) Error() string                 { return "PANIC: " + pe.err.Error() }
func (pe panicError) Unwrap() error                 { return pe.err } // Go 1.13 std "errors"
func (pe panicError) Cause() error                  { return pe.err } // "github.com/pkg/errors"
func (pe panicError) StackTrace() errors.StackTrace { return pe.err.StackTrace()[1:] }
func (pe panicError) Format(s fmt.State, verb rune) {
	switch verb {
	case 'v':
		_, _ = io.WriteString(s, "PANIC: ")
		if s.Flag('+') {
			fmt.Fprintf(s, "%v", pe.err)
			pe.StackTrace().Format(s, verb)
			return
		}
		_, _ = io.WriteString(s, pe.err.Error())
	case 's':
		_, _ = io.WriteString(s, pe.Error())
	case 'q':
		_, _ = fmt.Fprintf(s, "%q", pe.Error())
	}
}

var _ unwrapper = panicError{}
var _ causer = panicError{}
var _ featurefulError = panicError{}

// PanicToError takes an arbitrary object returned from recover(), and
// returns an appropriate error.
//
// If the input is nil, then nil is returned.
//
// If the input is an error returned from a previus call to
// PanicToError(), then it is returned verbatim.
//
// If the input is an error, it is wrapped with the message "PANIC:"
// and has a stack trace attached to it.
//
// If the input is anything else, it is formatted with "%+v" and
// returned as an error with a stack trace attached.
func PanicToError(rec interface{}) error {
	if rec == nil {
		return nil
	}
	switch rec := rec.(type) {
	case panicError:
		return rec
	case error:
		return panicError{err: errors.WithStack(rec).(featurefulError)}
	default:
		return panicError{err: errors.Errorf("%+v", rec).(featurefulError)}
	}
}
